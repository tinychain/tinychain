package core

import (
	"tinychain/db"
	"github.com/hashicorp/golang-lru"
	"tinychain/core/types"
	"tinychain/consensus"
	"tinychain/common"
	"sync/atomic"
	"sync"
	"math/big"
	"tinychain/event"
	"errors"
	"fmt"
	"sort"
)

var (
	cacheSize = 65535
	log       = common.GetLogger("blockchain")
)

// Blockchain is the canonical chain given a database with a genesis block
type Blockchain struct {
	db        *db.TinyDB       // chain db
	genesis   *types.Block     // genesis block
	lastBlock atomic.Value     // last block of chain
	engine    consensus.Engine // consensus engine
	event     *event.TypeMux   // eventhub
	quitCh    chan struct{}
	// TODO more fields

	dirtyBlk    sync.Map   // dirty block map
	dirtyHdr    sync.Map   // dirty header map
	blocksCache *lru.Cache // blocks lru cache
	headerCache *lru.Cache // headers lru cache

	appenBlkSub event.Subscription
}

func NewBlockchain(db *db.TinyDB, engine consensus.Engine) (*Blockchain, error) {
	blocksCache, _ := lru.New(cacheSize)
	headerCache, _ := lru.New(cacheSize)
	bc := &Blockchain{
		db:          db,
		engine:      engine,
		blocksCache: blocksCache,
		headerCache: headerCache,
		event:       event.GetEventhub(),
		quitCh:      make(chan struct{}, 1),
	}
	if err := bc.loadLastState(); err != nil {
		log.Errorf("failed to load last state from db, err:%s", err)
		return nil, err
	}
	return bc, nil
}

func (bc *Blockchain) Start() error {
	bc.appenBlkSub = bc.event.Subscribe(&AppendBlockEvent{})

	go bc.listen()
	return nil
}

func (bc *Blockchain) listen() {
	for {
		select {
		case ev := <-bc.appenBlkSub.Chan():
			blocks := ev.(*AppendBlockEvent).Blocks
			go func() {
				err := bc.AddBlocks(blocks)
				if err != nil {
					log.Error(err.Error())
				}
			}()
		case <-bc.quitCh:
			bc.appenBlkSub.Unsubscribe()
			return
		}
	}
}

func (bc *Blockchain) Stop() {
	close(bc.quitCh)
}

// loadLastState load the latest state of blockchain
func (bc *Blockchain) loadLastState() error {
	lastBlock, err := bc.db.GetLastBlock()
	if err != nil {
		// Should create genensis block
		return err
	}
	bc.lastBlock.Store(lastBlock)
	bc.blocksCache.Add(lastBlock.Hash(), lastBlock)
	// TODO

	return nil
}

func (bc *Blockchain) LastBlock() *types.Block {
	if block := bc.lastBlock.Load(); block != nil {
		return block.(*types.Block)
	}
	block, err := bc.db.GetLastBlock()
	if err != nil {
		log.Errorf("failed to get block from db, err:%s", err)
		return nil
	}
	return block
}

func (bc *Blockchain) GetBlock(hash common.Hash, height *big.Int) *types.Block {
	block, err := bc.db.GetBlock(height, hash)
	if err != nil {
		log.Errorf("failed to get block from db, err:%s", err)
		return nil
	}
	bc.blocksCache.Add(hash, block)
	return block
}

func (bc *Blockchain) GetBlockByHash(hash common.Hash) *types.Block {
	if block, ok := bc.blocksCache.Get(hash); ok {
		return block.(*types.Block)
	}
	height, err := bc.db.GetHeight(hash)
	if err != nil {
		log.Errorf("failed to get height by hash from db, err:%s", err)
		return nil
	}

	return bc.GetBlock(hash, height)
}

func (bc *Blockchain) GetHeaderByHash(hash common.Hash) (*types.Header, error) {
	if header, ok := bc.headerCache.Get(hash); ok {
		return header.(*types.Header), nil
	}
	height, err := bc.db.GetHeight(hash)
	if err != nil {
		return nil, err
	}
	header, err := bc.db.GetHeader(height, hash)
	if err != nil {
		return nil, err
	}
	bc.headerCache.Add(hash, header)
	return header, nil
}

func (bc *Blockchain) AddBlocks(blocks types.Blocks) error {
	for _, block := range blocks {
		if err := bc.AddBlock(block); err != nil {
			log.Errorf("failed to add block %s, err:%s", block.Hash(), err)
			continue
		}
	}

	valid, err := bc.Commit()
	if len(valid) > 0 {
		for _, height := range valid {
			bc.dirtyBlk.Delete(height)
		}
		go bc.event.Post(&BlockCommitEvent{
			Heights: valid,
		})
	}

	return err
}

// AddBlock appends block into chain.
// The blocks passed have been validated by block_pool.
func (bc *Blockchain) AddBlock(block *types.Block) error {
	if blk := bc.GetBlockByHash(block.Hash()); blk != nil {
		return errors.New(fmt.Sprintf("block %s exists in blockchain", blk.Hash().Hex()))
	}
	bc.dirtyBlk.Store(block.Height(), block)
	return nil
}

// Commit the blockchain to db.
// It returns valid block height and error if any.
func (bc *Blockchain) Commit() (valid []*big.Int, err error) {
	var (
		blocks types.Blocks
		last   = bc.LastBlock()
	)
	bc.dirtyBlk.Range(func(key, value interface{}) bool {
		blocks = append(blocks, value.(*types.Block))
		return true
	})

	sort.Sort(blocks)
	for _, block := range blocks {
		// Check block height equals to last height+1 or not
		if block.Height().Cmp(new(big.Int).Add(last.Height(), new(big.Int).SetInt64(1))) != 0 {
			return nil, errors.New(fmt.Sprintf(
				"block #%d cannot be added into blockchain, because current height of chain is %d",
				block.Height(), last.Height()))
		}
		// Commit block to db
		err = bc.db.PutBlock(block)
		if err != nil {
			log.Errorf("failed to commit block %s to db, err:%s", block.Hash(), err)
			break
		}
		last = block
		bc.blocksCache.Add(block.Height(), block)
		bc.dirtyBlk.Delete(block.Height())

		// Collect the valid block height
		valid = append(valid, block.Height())
	}
	if last != bc.LastBlock() {
		bc.lastBlock.Store(last)
	}
	return
}

func (bc *Blockchain) Engine() consensus.Engine {
	return bc.engine
}
