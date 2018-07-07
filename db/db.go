package db

import (
	"tinychain/common"
	"math/big"
	"tinychain/core/types"
	"tinychain/db/leveldb"
)

/*
	** Hash is block header hash


	"LastHeader" => the latest block header
	"LastBlock" => the latest block
	"WorldState" => the latest world state root hash

	"h" + block height + "n" => block hash
	"h" + block height + block hash => header
	"H" + block hash => block height
	"b" + block height + block hash => block
	"r" + block height + block hash => block receipts
	"l" + txHash => transaction meta data {hash,height,txIndex}
*/

const (
	KeyLastHeader = "LastHeader"
	KeyLastBlock  = "LastBlock"
	KeyWorldState = "WorldState"
)

var (
	log = common.GetLogger("tinydb")
)

// TinyDB stores and manages blockchain data
type TinyDB struct {
	db *leveldb.LDBDatabase
}

func NewTinyDB(db *leveldb.LDBDatabase) *TinyDB {
	return &TinyDB{db}
}

func (tdb *TinyDB) LDB() *leveldb.LDBDatabase {
	return tdb.db
}

func (tdb *TinyDB) GetWorldState() (common.Hash, error) {
	data, err := tdb.db.Get([]byte(KeyWorldState))
	if err != nil {
		return common.Hash{}, err
	}
	return common.BytesToHash(data), nil
}

func (tdb *TinyDB) GetLastBlock() (*types.Block, error) {
	data, err := tdb.db.Get([]byte(KeyLastBlock))
	if err != nil {
		return nil, err
	}
	block := &types.Block{}
	block.Deserialize(data)
	return block, nil
}

func (tdb *TinyDB) PutLastBlock(block *types.Block) error {
	data, _ := block.Serialize()
	err := tdb.db.Put([]byte(KeyLastBlock), data)
	if err != nil {
		return err
	}
	return nil
}

func (tdb *TinyDB) GetLastHeader() (*types.Header, error) {
	data, err := tdb.db.Get([]byte(KeyLastHeader))
	if err != nil {
		return nil, err
	}
	header := &types.Header{}
	header.Desrialize(data)
	return header, nil
}

func (tdb *TinyDB) PutLastHeader(header *types.Header) error {
	data, _ := header.Serialize()
	err := tdb.db.Put([]byte(KeyLastHeader), data)
	if err != nil {
		return err
	}
	return nil
}

func (tdb *TinyDB) GetHash(height *big.Int) (common.Hash, error) {
	var hash common.Hash
	data, err := tdb.db.Get([]byte("h" + height.String() + "n"))
	if err != nil {
		return hash, err
	}
	hash = common.DecodeHash(data)
	return hash, nil
}

func (tdb *TinyDB) PutHash(height *big.Int, hash common.Hash) error {
	err := tdb.db.Put([]byte("h"+height.String()+"n"), hash[:])
	if err != nil {
		return err
	}
	return nil
}

func (tdb *TinyDB) GetHeader(height *big.Int, hash common.Hash) (*types.Header, error) {
	data, err := tdb.db.Get([]byte("h" + height.String() + hash.String()))
	if err != nil {
		return nil, err
	}
	header := types.Header{}
	header.Desrialize(data)
	return &header, nil
}

func (tdb *TinyDB) PutHeader(header *types.Header) error {
	data, _ := header.Serialize()
	err := tdb.db.Put([]byte("h"+header.Height.String()+header.Hash().String()), data)
	if err != nil {
		return err
	}
	return nil
}

//// Total difficulty
//func (tdb *TinyDB) GetTD(height *big.Int, hash common.Hash) (*big.Int, error) {
//	data, err := tdb.db.Get([]byte("h" + height.String() + hash.String() + "t"))
//	if err != nil {
//		log.Errorf("Cannot find total difficulty with height %s and hash %s", height, hash)
//		return nil, err
//	}
//	return new(big.Int).SetBytes(data), nil
//
//}
//func (tdb *TinyDB) PutTD(height *big.Int, hash common.Hash, td *big.Int) error {
//	err := tdb.db.Put([]byte("h"+height.String()+hash.String()+"t"), td.Bytes())
//	if err != nil {
//		log.Errorf("Failed to put total difficulty with height %s and hash %s", height, hash)
//		return err
//	}
//	return nil
//}

func (tdb *TinyDB) GetHeight(hash common.Hash) (*big.Int, error) {
	data, err := tdb.db.Get([]byte("H" + hash.String()))
	if err != nil {
		return nil, err
	}
	return new(big.Int).SetBytes(data), nil
}

func (tdb *TinyDB) PutHeight(hash common.Hash, height *big.Int) error {
	err := tdb.db.Put([]byte("H"+hash.String()), height.Bytes())
	if err != nil {
		return err
	}
	return nil
}

func (tdb *TinyDB) GetBlock(height *big.Int, hash common.Hash) (*types.Block, error) {
	data, err := tdb.db.Get([]byte("b" + height.String() + hash.String()))
	if err != nil {
		return nil, err
	}
	block := types.Block{}
	block.Deserialize(data)
	return &block, nil
}

func (tdb *TinyDB) PutBlock(block *types.Block) error {
	height := block.Header.Height
	hash := block.Header.Hash()
	data, _ := block.Serialize()
	err := tdb.db.Put([]byte("b"+height.String()+hash.String()), data)
	if err != nil {
		return err
	}
	return nil
}

func (tdb *TinyDB) GerReceipts(height *big.Int, hash common.Hash) (types.Receipts, error) {
	data, err := tdb.db.Get([]byte("r" + height.String() + hash.String()))
	if err != nil {
		return nil, err
	}
	var receipts types.Receipts
	err = receipts.Deserialize(data)
	if err != nil {
		return nil, err
	}
	return receipts, nil
}

func (tdb *TinyDB) PutReceipts(height *big.Int, hash common.Hash, receipt types.Receipt) error {
	data, err := receipt.Serialize()
	if err != nil {
		return err
	}
	return tdb.db.Put([]byte("r"+height.String()+hash.String()), data)
}

func (tdb *TinyDB) GetTxMeta(txHash common.Hash) (*types.TxMeta, error) {
	data, err := tdb.db.Get([]byte("l" + txHash.String()))
	if err != nil {
		log.Errorf("Cannot find txMeta with txHash %s", txHash.Hex())
		return nil, err
	}
	txMeta := &types.TxMeta{}
	txMeta.Deserialize(data)
	return txMeta, nil
}

func (tdb *TinyDB) PutTxMetaInBatch(block *types.Block) error {
	batch := tdb.db.NewBatch()
	for i, tx := range block.Transactions {
		txMeta := &types.TxMeta{
			Hash:    block.Hash(),
			Height:  block.Height(),
			TxIndex: uint64(i),
		}
		data, _ := txMeta.Serialize()
		batch.Put([]byte("l"+tx.Hash().String()), data)
	}
	return batch.Write()
}
