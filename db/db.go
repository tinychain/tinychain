package db

import (
	"tinychain/common"
	"tinychain/core/types"
	"tinychain/db/leveldb"
	"strconv"
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

func (tdb *TinyDB) GetLastBlock() (common.Hash, error) {
	data, err := tdb.db.Get([]byte(KeyLastBlock))
	if err != nil {
		return common.Hash{}, err
	}
	return common.BytesToHash(data), nil
}

func (tdb *TinyDB) PutLastBlock(hash common.Hash) error {
	return tdb.db.Put([]byte(KeyLastBlock), hash.Bytes())
	return nil
}

func (tdb *TinyDB) GetLastHeader() (common.Hash, error) {
	data, err := tdb.db.Get([]byte(KeyLastHeader))
	if err != nil {
		return common.Hash{}, err
	}
	return common.BytesToHash(data), nil
}

func (tdb *TinyDB) PutLastHeader(hash common.Hash) error {
	err := tdb.db.Put([]byte(KeyLastHeader), hash.Bytes())
	if err != nil {
		return err
	}
	return nil
}

func (tdb *TinyDB) GetHash(height uint64) (common.Hash, error) {
	var hash common.Hash
	data, err := tdb.db.Get([]byte("h" + strconv.FormatUint(height, 10) + "n"))
	if err != nil {
		return hash, err
	}
	hash = common.DecodeHash(data)
	return hash, nil
}

func (tdb *TinyDB) PutHash(height uint64, hash common.Hash) error {
	err := tdb.db.Put([]byte("h"+strconv.FormatUint(height, 10)+"n"), hash[:])
	if err != nil {
		return err
	}
	return nil
}

func (tdb *TinyDB) GetHeader(height uint64, hash common.Hash) (*types.Header, error) {
	data, err := tdb.db.Get([]byte("h" + strconv.FormatUint(height, 10) + hash.String()))
	if err != nil {
		return nil, err
	}
	header := types.Header{}
	header.Desrialize(data)
	return &header, nil
}

func (tdb *TinyDB) PutHeader(header *types.Header) error {
	data, _ := header.Serialize()
	err := tdb.db.Put([]byte("h"+strconv.FormatUint(header.Height, 10)+header.Hash().String()), data)
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

func (tdb *TinyDB) GetHeight(hash common.Hash) (uint64, error) {
	data, err := tdb.db.Get([]byte("H" + hash.String()))
	if err != nil {
		return 0, err
	}
	return common.Bytes2Uint(data), nil
}

func (tdb *TinyDB) PutHeight(batch *leveldb.Batch, hash common.Hash, height uint64, sync, flush bool) error {
	batch.Put([]byte("H"+hash.String()), common.Uint2Bytes(height))
	if flush {
		if sync {
			return batch.Write()
		} else {
			go batch.Write()
		}
	}
	return nil
}

func (tdb *TinyDB) GetBlock(height uint64, hash common.Hash) (*types.Block, error) {
	data, err := tdb.db.Get([]byte("b" + strconv.FormatUint(height, 10) + hash.String()))
	if err != nil {
		return nil, err
	}
	block := types.Block{}
	block.Deserialize(data)
	return &block, nil
}

func (tdb *TinyDB) PutBlock(batch *leveldb.Batch, block *types.Block, sync, flush bool) error {
	height := block.Height()
	hash := block.Hash()
	data, _ := block.Serialize()
	batch.Put([]byte("b"+strconv.FormatUint(height, 10)+hash.String()), data)
	if flush {
		if sync {
			return batch.Write()
		} else {
			go batch.Write()
		}
	}
	return nil
}

func (tdb *TinyDB) GerReceipts(height uint64, hash common.Hash) (types.Receipts, error) {
	data, err := tdb.db.Get([]byte("r" + strconv.FormatUint(height, 10) + hash.String()))
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

func (tdb *TinyDB) PutReceipts(batch *leveldb.Batch, height uint64, hash common.Hash, receipts types.Receipts, sync, flush bool) error {
	data, err := receipts.Serialize()
	if err != nil {
		return err
	}
	batch.Put([]byte("r"+strconv.FormatUint(height, 10)+hash.String()), data)
	if flush {
		if sync {
			return batch.Write()
		} else {
			go batch.Write()
		}
	}
	return nil
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

// PutTxMetas put transactions' meta to db in batch
func (tdb *TinyDB) PutTxMetas(batch *leveldb.Batch, txs types.Transactions, hash common.Hash, height uint64, sync, flush bool) error {
	for i, tx := range txs {
		txMeta := &types.TxMeta{
			Hash:    hash,
			Height:  height,
			TxIndex: uint64(i),
		}
		data, _ := txMeta.Serialize()
		if err := batch.Put([]byte("l"+tx.Hash().String()), data); err != nil {
			return err
		}
	}
	if flush {
		if sync {
			return batch.Write()
		} else {
			go batch.Write()
		}
	}
	return nil
}
