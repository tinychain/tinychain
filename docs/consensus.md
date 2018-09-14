# Design
In tinychain, consensus module is designed as **pluggable pattern**. Developers can change consensus algorithm to meet different scene easily。

We provided several consensus algorithm in advance:

## 1. Solo
Solo is an **MVP** consensus engine. At this mechanism, only one block producer exists in the network, and other peers trust this one and receive all the blocks it proposes.

Using **Solo** engine, tinychain degenerates as a centralized distributed ledger system.

Not-BP peers receive blocks and process:

1. Verify block header. If valid, go on next processing, and broadcast blocks to other peers at the same time.
2. Execute the transactions in the block, and record `receipts` and compute current `state_root`.
3. Compare the hash of transactions and receipts, and `state_root` with those included in block header. If match, commit the block to database.

## 2. VRF + BFT
VRF(Verifiable Random Function)

WIP

## 3. POW
Proof-of-work in Tinychain is a general implementation of consensus algorithm, inspired by bitcoin.

### Interval of block proposal
Every new block will be mined every 30 seconds in the network.

### Difficulty adjustment
Proof-of-work is actually to compute a difficult work violently. Base on the inequality `Hash(data,nonce) <  target`, miners travel through all possible nonce and try to compute the result of `Hash(data,nonce)`, and compare it with a `target`.

`target` is calculated with `target = MAXIMUM_TARGET / difficulty`. `MAXIMUM_TARGET`is a predefined limit(2^(256-32)), which is the same as bitcoin; `diffculty` is difficulty, stored in the field `consensus info` of blocks.

Difficulty will be adjusted every 20160 blocks mined. And the adjustment formula is `new_target = curr_target * actual duration of 20160 blocks / theoretical duration of 20160 blocks`

We need to care about some details in implementation:
- prevent boom of difficulty
    - if the last 20160 blocks take less than 1/4 week, then use 1/4 week as duration to prevent difficulty to increase by more than 4 times.
    - if the last 2016 blocks take larger than 4. week, then use 4. week as duration to prevent difficulty to decrease by more than 4 times.
- if the new difficulty is lower than the minimum limit(namely 1), use the minimum.
- if the new difficulty is larger than the maximum limit(namely 2^(256-32)), use the maximum. 

# Pluggable design
Consensus engine communicate with other modules through **event mechanism**.

## Block Pool
Block pool is used to collect new blocks from network, and validate block headers. If valid, it will add the block to the pool.

The block headers **will not be verified again** in consensus process.

## Subscribed Event Type
For clearer description, we classify events into those which peers that produce blocks and not produce blocks should subscribe.

### BP
- `core.ConsensusEvent` - `Executor` complete to execute and finalize new block, and notify subscriber。
- `core.ExecPendingTxEvent` - `tx_pool` collect enough transactions, and notify subscriber。

### Not-BP
- `core.BlockReadyEvent` - `block_pool` receives new blocks, and notify subscriber。
- `core.NewReceiptsEvent` - `Executor` executes blocks and retrieve `receipts`，and notify subscriber。

### Both
- `core.CommitCompleteEvent` - `Exector` complete to commit blocks to database，and notify subscriber。

## Post Event Type
In consensus process, consensus engine need to use other modules to process the info, such as executing blocks by `Executor`, or notifying network to transfer info, etc.

Some event types used by consensus engine are introduced below(can be more):

### BP
- `core.ProposeBlockEvent` - `Consensus` proposes new blocks, and posts it to `Executor` to execute and finalize blocks.
- `core.CommitBlockEvent` - notify `Executor` to commit blocks to database.
- `p2p.MulticastEvent` - notify network to multicast new blocks.

### Not-BP
- `core.ExecBlockEvent` - notify `Executor` to execute blocks, and retrieve `receipts` through `core.NewReceiptsEvent`.
- `p2p.BroadcastEvent` - notify network to broadcast blocks.

### Both
- `core.CommitBlockEvent` - notify `Executor` to commit blocks to database.

# Interface of components in consensus engine
```Go
type Blockchain interface {
    // for demand
    LastBlock() *types.Block
    ...
}

type BlockValidator interface {
	ValidateHeader(header *types.Header) error
	ValidateHeaders(headers []*types.Header) (chan struct{}, chan error)
	ValidateBody(b *types.Block) error
	ValidateState(b *types.Block, state *state.StateDB, receipts types.Receipts) error
}

type TxValidator interface{
	ValidateTx(transaction *types.Transaction) error
}
```