# 共识模块设计
在本项目中，共识模块被设计成可插拔的模式。开发者可自行替换共识算法，以适应不同需求的场景。

共识算法有以下几种选择——

## 1. Solo
Solo是一个最小可用的共识引擎。在该机制下，全网只有一个出块节点，其余节点**默认相信该出块节点是诚实的**，接收它所提案的区块并做验证。

在该共识机制下，区块链系统退化为中心化的分布式账本系统。

### 何时出块
Solo采用惰性出块的方式。当出块节点的交易池`tx_pool`接收到的交易达到某一预设上限，或者新交易驻留时间超过某一阈值时，则会提案新的区块。

其他节点在接收到区块后，执行如下流程：
1. 验证区块头是否合法；若合法，执行下一流程，同时广播给其他节点。
2. 执行区块中的交易，记录下交易收据`receipts`和当前状态`state_root`。
3. 比对交易数据的Hash和状态root是否与区块头所提供的数据相符。若相符，则提交区块到数据库。

## 2. VRF + BFT
VRF(Verifiable Random Function)

WIP

## 3. POW
Tinychain中的Proof-of-work是一个常规的工作量证明共识算法，灵感来源于比特币。
### 出块间隔
全网大约每30秒生成一个新区块。

### 难度调整
工作量证明的算法实际为暴力计算难题。基于公式`Hash(data,nonce) < target`，挖矿节点遍历所有的nonce情况，找出一个满足该不等式的nonce，并为区块签名。

`target`的计算公式为`target = MAXIMUM_TARGET / difficulty`。`MAXIMUN_TARGET`为一个预设的最大上限(2^(256-32))；`diffculty`为难度，保存在区块的`consensus info`字段。

每经过20160个区块（大约一周），进行一次难度调整。难度调整公式为`new_target = curr_target * 实际20160个块的出块时间 / 理论20160个块的出块时间（1周）`

在具体实现时，需要注意几个细节：
- 防止难度爆炸
    - 前20160个块用时小于1/4周时，则按1/4周算，防止难度增加4倍以上。
    - 前20160个块用时大于4周时，则按4周算，防止难度降低4倍以上。
- 如果低于最小难度限制（即为1），则按最小难度处理。
- 如果大于最大难度限制(2^(256-32)，则按最大难度处理。

# 可插拔设计
共识引擎与其他模块通过消息机制进行通信。

## 区块池(Block Pool)
区块池用于收集网络中的新区块，并验证区块头是否合法。若合法，则将该新区块加入到池中供共识引擎获取。

由于**区块池已预先做了区块头验证**，因此共识流程不再需要验证区块头。

## 订阅的事件类型
为了说明更清楚，故事件类型按出块与非出块节点进行区分。但在某些共识机制下，一个节点既可能是出块节点，也可能是非出块节点，所以**建议对以下所有事件类型均做监听并实现响应的处理逻辑**。
### 出块节点
- `core.ConsensusEvent` - `Executor`完成新区块的提案、执行和封装，通知订阅者。
- `core.ExecPendingTxEvent` - `tx_pool`收集到足够的交易后，通知订阅者。

### 非出块节点
- `core.BlockReadyEvent` - `block_pool`接收到新的新区块到达，则通知订阅者。
- `core.NewReceiptsEvent` - `Executor`执行区块后，发送新的交易回执`receipts`，通知订阅者。

### 所有类型节点
- `core.CommitCompleteEvent` - `Exector`成功提交区块至数据库后，通知订阅者。

## 对外发送的事件类型
共识流程中难免需要使用其他模块对共识信息进行处理，如将区块交由`Executor`执行，或通知网络层发送消息等。

以下是共识引擎需要发送（包含但不限于）的一些消息类型——

### 出块节点
- `core.ProposeBlockEvent` - 共识引擎提案了新的区块。该事件由`Executor`监听，封装成完成区块。
- `core.CommitBlockEvent` - 通知`Executor`提交区块至数据库。
- `p2p.MulticastEvent` - 通知网络层组播新区块。

### 非出块节点
- `core.ExecBlockEvent` - 通知`Executor`执行区块，并获取回执（通过`core.NewReceiptsEvent`)
- `p2p.BroadcastEvent` - 通知网络层广播区块。

### 所有类型节点
- `core.CommitBlockEvent` - 通知`Executor`提交区块至数据库。

# 共识引擎各组件的接口定义
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
