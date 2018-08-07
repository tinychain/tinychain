# WIP
This project is working in progress. A tiny blockchain

## Features
### Consensus
In consensus module, we design and implements different consensus algorithm, and make them **pluggable**.
 
please read [document](docs/consensus.md)

### Network
We use libp2p to implements network layer, including:

- peers discovering
- peers communication
    - blocks transferring
    - transactions transferring
    - consensus information transferring 

### Cryptography
We decide to use **Ed5519** cryptographic algorithm to identify peers and produce signature.

Ed25519 is a public-key signature system with several attractive features:

- Fast single-signature verification
- Even faster batch verification
- Very fast signing
- Fast key generation
- High security level
- Collision resilience
- so on...

### Database
LevelDB

### Merkle tree
In tinychain, we use **Bucket tree** to induce transactions and world state.

Bucket tree is a variant merkle tree with several features that are different from the common merkle tree:

1. fix height of tree when initialize and will not be changed by the amount of transactions at leaf nodes.
2. low-cost to recompute the root hash when add or remove a kv pair to/from tree.
3. customizable capacity and aggreation.

### Event Hub
Eventhub in tinychain is extended from `TypeMux` and `feed` in Ethereum. We combine them and re-implement a new event hub to achieve a better performance and readability, and make the processing flow clearer.