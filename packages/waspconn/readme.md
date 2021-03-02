# WaspConn dApp (plugin) for Goshimmer

## Purpose

The _WaspConn_ dApp handles connection with Wasp nodes, it is a Wasp proxy
in Goshimmer. 

One or several Wasp nodes can be connected to one Goshimmer node. The Wasp node
can only be connected to one Goshimmer node (this may change in the future).

WaspConn's `utxodb` feature is used for testing purposes (see below). Implemented 
`utxodb` API endpoints are also used just for testing purposes.   

## UTXODB
WaspConn contains `utxodb`package which contains in-memory, centralized (non-distributed) 
ledger of value transactions. 
`utxodb` allows testing of most of Wasp features without having access to distributed Value Tangle 
on Goshimmer network.

Features of `utxodb`:
 
- contains deterministically pre-defined genesis transaction with initial supply of IOTA tokens.
- validates value transactions, validates signatures agains addresses, rejects conflicting transactions, 
provides API to retrieve address balances and so on. In other words, maintains consistent UTXO ledger
- emulates confirmation delay and conflict handling strategies (configurable).
- exposes pre-defined and pre-loaded with tokens addresses and its signature schemes (private keys): 
the genesis address and 10 testing addresses. This is completely deterministic and is used for testing purposes.  

## Dependency

WaspConn is not dependent on Wasp. Instead, Wasp has Goshimmer with WaspConn 
dApp (`wasp` branch of the Goshimmer) as dependency.

WaspConn and Goshimmer are unaware about smart contract transactions. They treat it as just
ordinary value transaction with data payloads. 

## Protocol

WaspConn implements its part of the protocol between Wasp node and Goshimmer. 

- The protocol is completely **asynchronous messaging**: neither party is waiting for the response or 
confirmation after sending message to another party.  
Even if a message is request for example to get a transaction, the Wasp node receives
response asynchronously. 
It also means, that messages may be lost without notification.

- the transport between Goshimmer and Wasp is using `BufferedConnection` provided by `hive.go`. 
Protocol can handle practically unlimited message sizes.

Functions:

#### Posting a transaction
Wasp node may post the transaction to Goshimmer for confirmation just like any other external program, for example 
a wallet. 
If `utxodb` is enabled, the transaction goes right into the confirmation emulation mechanism of `utxodb`. 

#### Subscription
Wasp node subscribes to transaction it wants to receive. It sends a list of addresses of smart contracts 
it is running and WaspConn is sending to Wasp any new confirmed transaction which has subscribed address among it 
outputs.

#### Requesting a transaction
Wasp may request a transaction by hash. WaspConn plugin sends the confirmed transaction to Wasp (if found).

#### Requesting address balances
Wasp may request address balances from Goshimmer. 
WaspConn sends confirmed UTXOs which belongs to the address.  

#### Sending request backlog to Wasp
Upon request, WaspCon may send not only UTXO contained in the address, but it may also analyse colored 
tokens and send origin transactions of corresponding colors if they contain unspent outputs. 
This is how backlog of requests is handled on the tangle.

## Configuration
All configuration values for the WaspConn plugin are in the `waspconn` portion of the `config.json` file.
```
  "waspconn": {
    "port": 5000,
    "utxodbenabled": true,
    "utxodbconfirmseconds": 0,
    "utxodbconfirmrandomize": false,
    "utxodbconfirmfirst": true
  }
```
- `waspconn.port` specifies port where WaspCon is listening for new Wasp connections.
- `waspconn.utxodbenabled` is a boolean flag which specifies if WaspConn is mocking the value tangle (`true`) or it 
is using the ValueTangle provided by Goshimmer.

If `waspconn.utxodbenabled: true` the following values configure parameters of mocked of the Value Tangle:

- `waspconn.utxodbconfirmseconds` specifies emulated confirmation time. When new transaction is posted, 
specified amount of seconds it is in `pending` state, and only after the confirmation time it is included 
into the `utxodb` ledger.
`0` seconds means it is included immediately and result is known to the posting call synchronously.
- `waspconn.utxodbconfirmrandomize` if `false`, the emulated confirmation time is constant, 
if `true` it is uniformly distributed around the confirmation time parameter.
-  `waspconn.utxodbconfirmfirst` determines behavior in case of conflicting transactions. If `true`, 
eventually the first out of all conflicting transactions will be included into `utxodb` ledger. If `false` 
all conflicting transactions will be rejected within duration period between posting first of them and 
the supposed confirmation deadline.
   
