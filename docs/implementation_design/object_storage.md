# Object storage

## Database
GoShimmer allows to store data in a persistent way or keeps it in memory, depending on provided init parameter `CfgDatabaseInMemory` value. In memory storage is purely based on Go map. For the persistent databes it uses `badger` from hive.go package. It is simple and fast key-value database that performs well for both reads and writes simultaneously.  

Database plugin is responsible for creating `store` object. It will manage proper closure of the database upon recieving shutdown signal. During the start configuration, database is marked as unhealthy, it will be marked as healthy on shutdown, then garbage collector is run and database is closed.

## ObjectStorage
ObjectStorage is a manual cache which keeps objects as long as consumers are using it. It is used as a base data structure for many data collections elements such as `branchStorage`, `conflictStorage`, `messageStorage` and many others.
It uses key-value storage type, and allows store catched object, define its object factory, and mutex options for guarding shared variables and preventing changing object state by multiple gorutines at the same time.
It takes care of  dynamic creation of different object types depending on the key and the serialized data it recieves.
## Tangle
The most important data collection is the tangle struct defined in `tangle` package. This object connects all different components like: `solidifier, requester, scheduler,...` responsible for message handlig and validating.
It is a central data structure of IOTA protocol.
  
```Go
type Tangle struct {
   	Parser         *Parser
   	Storage        *Storage
   	Solidifier     *Solidifier
   	Scheduler      *Scheduler
   	Booker         *Booker
   	TipManager     *TipManager
   	Requester      *Requester
   	MessageFactory *MessageFactory
   	LedgerState    *LedgerState
   	Utils          *Utils
   	Options        *Options
   	Events         *Events
   
   	OpinionFormer            *OpinionFormer
   	PayloadOpinionProvider   OpinionVoterProvider
   	TimestampOpinionProvider OpinionProvider
   
   	setupParserOnce sync.Once
   }
```
All configurable parameters for Tangle are stored in `Options` attribute and allows to specify: `Store` database instance,  node's `Identity` or `TangleWidth` which allows to artificially change number of tips in the tangle.  
Components of the Tangle responsible for storing data: 
  - `Storage` - representing storage of the messages
  - `LedgerState` - wrapper for all components from `ledgerstate` package that helps to track the state of the ledger and makes them available at a "single point of contact".
  - `TipManager` - manages a map of tips and emits events for their removal and addition.

The Tangle structure is used in the messagelayer plugin, it initiates only one instance of the Tangle. During configuration is starts the configuration of all its components and if the file exists it is loading snapshot containing the state of the ledger to `ledgerstate` attribute. 

### Storage 
`Storage` data structure consists from
 - `messageStorage` - 
 - `messageMetadataStorage` - 
 - `approverStorage` - 
 - `missingMessageStorage` - 
 - `attachmentStorage` - 
 - `markerIndexBranchIDMappingStorage` - 
 

New messages can be added to the tangle with the `StoreMessage()` function. In current implementation this function can be triggered by one of two events: 
  - MessageParsed - emited by `Parser` component
  - MessageConstructed - emited by `MeassageFactory`after creation of a new message
  
### Ledger state
The state of the ledger is described with two components: `branchDAG` and `utxoDAG`.

### Tip manager