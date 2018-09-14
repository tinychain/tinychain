package chain

import (
	"errors"
	"fmt"
	"github.com/hashicorp/golang-lru"
	"sync"
	"sync/atomic"
	"tinychain/common"
	"tinychain/core/types"
	tdb "tinychain/db"
	"tinychain/db/leveldb"
)

var (
	log = common.GetLogger("blockchain")

	errGenesisExist = errors.New("genesis has existed")
)

const (
	blockCacheLimit  = 1024
	headerCacheLimit = 2048
)

// Blockchain is the canonical chain given a database with a genesis block
type Blockchain struct {
	db             *tdb.TinyDB  // chain db
	genesis        *types.Block // genesis block
	lastBlock      atomic.Value // last block of chain
	lastFinalBlock atomic.Value // last final block of chian
	mu             sync.RWMutex

	blockCache  *lru.Cache // blocks lru cache
	headerCache *lru.Cache // headers lru cache
}

func NewBlockchain(db *leveldb.LDBDatabase) (*Blockchain, error) {
	blockCache, _ := lru.New(blockCacheLimit)
	headerCache, _ := lru.New(headerCacheLimit)
	bc := &Blockchain{
		db:          tdb.NewTinyDB(db),
		blockCache:  blockCache,
		headerCache: headerCache,
	}
	bc.genesis = bc.GetBlockByHeight(0)
	initChain(bc)
	return bc, nil
}

func (bc *Blockchain) Genesis() *types.Block {
	return bc.genesis
}

func (bc *Blockchain) SetGenesis(genesis *types.Block) error {
	if bc.genesis != nil {
		return errGenesisExist
	}
	bc.genesis = genesis
	if err := bc.AddBlock(genesis); err != nil {
		return err
	}

	if err := bc.db.PutBlock(tdb.GetBatch(bc.db.LDB(), 0), genesis, true, true); err != nil {
		log.Errorf("failed to persist genesis, %s", err)
		return err
	}
	return nil
}

// Reset init blockchain with genesis block
func (bc *Blockchain) Reset() error {
	return bc.ResetWithGenesis(bc.genesis)
}

func (bc *Blockchain) ResetWithGenesis(genesis *types.Block) error {
	batch := bc.db.LDB().NewBatch()
	if err := bc.db.PutBlock(batch, genesis, false, false); err != nil {
		log.Errorf("failed to put genesis into db, err:%s", err)
		return err
	}
	bc.mu.Lock()
	defer bc.mu.Unlock()
	if err := bc.db.PutLastBlock(batch, genesis.Hash(), false, false); err != nil {
		log.Errorf("failed to put genesis hash into db, err:%s", err)
		return err
	}
	if err := batch.Write(); err != nil {
		log.Errorf("failed to commit batch to db, err:%s", err)
		return err
	}
	bc.Purge()
	bc.blockCache.Add(genesis.Height(), genesis)
	bc.genesis = genesis
	bc.lastBlock.Store(genesis)
	return nil
}

// Purge drop the blocks in memory and revert the blockchain to the height of `lastFinalBlock`
func (bc *Blockchain) Purge() {
	bc.mu.Lock()
	defer bc.mu.Unlock()

	bc.lastBlock.Store(nil)
	bc.blockCache.Purge()
	bc.headerCache.Purge()
}

func (bc *Blockchain) LastHeight() uint64 {
	return bc.LastBlock().Height()
}

// LastBlock returns the last block of latest blockchain in memory
func (bc *Blockchain) LastBlock() *types.Block {
	if block := bc.lastBlock.Load(); block != nil {
		return block.(*types.Block)
	}
	block := bc.LastFinalBlock()
	bc.lastBlock.Store(block)
	return block
}

// LastFinalBlock returns the last commited block in db
func (bc *Blockchain) LastFinalBlock() *types.Block {
	if fb := bc.lastFinalBlock.Load(); fb != nil {
		return fb.(*types.Block)
	}
	hash, err := bc.db.GetLastBlock()
	if err != nil {
		panic(fmt.Sprintf("failed to get last block's hash from db, err:%s", err))
	}
	block := bc.GetBlockByHash(hash)
	bc.lastFinalBlock.Store(block)
	return block
}

func (bc *Blockchain) GetHeader(hash common.Hash, height uint64) *types.Header {
	if header, ok := bc.headerCache.Get(hash); ok {
		return header.(*types.Header)
	}
	header, err := bc.db.GetHeader(height, hash)
	if err != nil {
		return nil
	}
	bc.headerCache.Add(hash, header)
	return header
}

func (bc *Blockchain) GetBlock(hash common.Hash, height uint64) *types.Block {
	block, err := bc.db.GetBlock(height, hash)
	if err != nil {
		log.Errorf("failed to get block from db, err:%s", err)
		return nil
	}
	bc.blockCache.Add(hash, block)
	return block
}

func (bc *Blockchain) GetBlockByHeight(height uint64) *types.Block {
	hash, err := bc.db.GetHash(height)
	if err != nil {
		log.Errorf("failed to get hash from db, err:%s", err)
		return nil
	}
	return bc.GetBlock(hash, height)
}

func (bc *Blockchain) GetBlockByHash(hash common.Hash) *types.Block {
	if block, ok := bc.blockCache.Get(hash); ok {
		return block.(*types.Block)
	}
	height, err := bc.db.GetHeight(hash)
	if err != nil {
		log.Errorf("failed to get height by hash from db, err:%s", err)
		return nil
	}
	return bc.GetBlock(hash, height)
}

func (bc *Blockchain) GetHeaderByHash(hash common.Hash) *types.Header {
	height, err := bc.db.GetHeight(hash)
	if err != nil {
		return nil
	}
	if header, ok := bc.headerCache.Get(hash); ok {
		return header.(*types.Header)
	}
	header, err := bc.db.GetHeader(height, hash)
	if err != nil {
		return nil
	}
	bc.headerCache.Add(hash, header)
	return header
}

// AddBlocks insert blocks in batch when importing outer blockchain
func (bc *Blockchain) AddBlocks(blocks types.Blocks) error {
	for _, block := range blocks {
		if err := bc.AddBlock(block); err != nil {
			log.Errorf("failed to add block %s, err:%s", block.Hash(), err)
			return err
		}
	}

	return nil
}

// AddBlock appends block into chain.
// The blocks passed have been validated by block_pool.
func (bc *Blockchain) AddBlock(block *types.Block) error {
	if blk := bc.GetBlockByHash(block.Hash()); blk != nil {
		return errors.New(fmt.Sprintf("block %s exists in blockchain", blk.Hash().Hex()))
	}
	last := bc.LastBlock()
	// Check block height equals to last height+1 or not
	if block.Height() != last.Height()+1 {
		return errors.New(fmt.Sprintf(
			"block #%d cannot be added into blockchain because its previous block height is #%d",
			block.Height(), last.Height()))
	}
	bc.blockCache.Add(block.Hash(), block)
	bc.headerCache.Add(block.Hash(), block.Header)
	bc.lastBlock.Store(block)
	return nil
}

// commit persist the block to db.
func (bc *Blockchain) CommitBlock(batch *leveldb.Batch, block *types.Block) error {
	// Put block to db.Batch
	if err := bc.db.PutBlock(batch, block, false, false); err != nil {
		log.Errorf("failed to put block %s in db, err:%s", block.Hash(), err)
	}
	if err := bc.db.PutLastBlock(batch, block.Hash(), false, false); err != nil {
		log.Errorf("failed to put last block hash %s to db, err:%s", block.Hash(), err)
		return err
	}
	if err := bc.db.PutHeader(batch, block.Header, false, false); err != nil {
		log.Errorf("failed to put header %s in db, err:%s", block.Hash(), err)
		return err
	}
	if err := bc.db.PutLastHeader(batch, block.Hash(), false, false); err != nil {
		log.Errorf("failed to put header hash %s in db, err:%s", block.Hash(), err)
		return err
	}
	bc.lastFinalBlock.Store(block)
	return nil
}
