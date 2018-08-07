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
### BP
- `core.ProposeBlockEvent` - `Consensus` proposes new blocks, and posts it to `Executor` to execute and finalize blocks.
- `core.CommitBlockEvent` - notify `Executor` to commit blocks to database.
- `p2p.MulticastEvent` - notify network to multicast new blocks.

### Not-BP
- `core.ExecBlockEvent` - notify `Executor` to execute blocks, and retrieve `receipts` through `core.NewReceiptsEvent`.
- `p2p.BroadcastEvent` - notify network to broadcast blocks.

### Both
- `core.CommitBlockEvent` - notify `Executor` to commit blocks to database.
