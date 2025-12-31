# WarpCustomVM Architecture Diagrams

This document contains visual architecture diagrams in Mermaid format. These can be viewed in any Markdown viewer that supports Mermaid (GitHub, GitLab, VS Code with extensions, etc.).

## Table of Contents
1. [Component Architecture](#component-architecture)
2. [Outgoing Message Flow](#outgoing-message-flow)
3. [Incoming Message Flow](#incoming-message-flow)
4. [Consensus Synchronization](#consensus-synchronization)
5. [Data Flow](#data-flow)
6. [State Management](#state-management)

---

## Component Architecture

```mermaid
graph TB
    subgraph "External Clients"
        USER[User/Application]
        RELAYER[ICM Relayer]
        CCHAIN[C-Chain Contract]
    end

    subgraph "WarpCustomVM Instance"
        subgraph "API Layer"
            RPC[JSON-RPC Server]
            SUBMIT[submitMessage]
            RECEIVE[receiveWarpMessage]
            GETALL[getAllReceivedMessages]
            GETBLOCK[getLatestBlock/getBlock]
        end

        subgraph "Core Components"
            BUILDER[Builder<br/>Block Creation]
            CHAIN[Chain<br/>Consensus & Accept]
            STATE[State<br/>Database Persistence]
        end

        subgraph "Warp Integration"
            ACP118[ACP-118 Handler<br/>Signature Requests]
            VERIFIER[Warp Verifier<br/>Message Verification]
            P2P[P2P Network<br/>Validator Communication]
        end
    end

    subgraph "Consensus Layer"
        ENGINE[Snowman Consensus Engine]
        VALIDATORS[Other Validators]
    end

    subgraph "Storage"
        DB[(LevelDB/PebbleDB)]
    end

    USER -->|1. Submit Message| RPC
    CCHAIN -->|2. Send Warp Message| RELAYER
    RELAYER -->|3. Deliver Message| RPC
    
    RPC --> SUBMIT
    RPC --> RECEIVE
    RPC --> GETALL
    RPC --> GETBLOCK
    
    SUBMIT -->|Create Message| BUILDER
    RECEIVE -->|Parse & Verify| BUILDER
    GETALL -->|Query| STATE
    GETBLOCK -->|Query| STATE
    
    BUILDER -->|Propose Block| ENGINE
    ENGINE -->|Accept Block| CHAIN
    CHAIN -->|Store State| STATE
    STATE -->|Read/Write| DB
    
    RELAYER <-->|Request Signatures| ACP118
    ACP118 <--> P2P
    P2P <--> VALIDATORS
    ACP118 --> VERIFIER
    VERIFIER --> STATE
    
    ENGINE <--> VALIDATORS

    style USER fill:#e1f5ff
    style RELAYER fill:#fff4e1
    style CCHAIN fill:#ffe1e1
    style RPC fill:#e8f5e9
    style BUILDER fill:#fff3e0
    style CHAIN fill:#f3e5f5
    style STATE fill:#e3f2fd
    style DB fill:#eceff1
```

---

## Outgoing Message Flow (WarpCustomVM → C-Chain)

```mermaid
sequenceDiagram
    participant User
    participant API as API Server
    participant Builder
    participant Chain
    participant State as Database
    participant Consensus as Validators
    participant Relayer as ICM Relayer
    participant CChain as C-Chain

    User->>API: submitMessage(destChain, destAddr, payload)
    API->>API: Create unsigned Warp message
    API->>API: Wrap in AddressedCall format
    API->>Builder: AddMessage(unsignedMsg)
    API-->>User: Return messageID

    Note over Builder: Wait for block time (1 sec)
    
    Builder->>Builder: BuildBlock()
    Builder->>Builder: Include pending messages
    Builder->>Consensus: Propose block
    
    Consensus->>Consensus: Vote on block
    Consensus->>Chain: Accept(block)
    
    Chain->>State: SetLatestBlockHeader(block)
    State->>State: Store block with messages
    
    Note over State: Block now contains message

    Relayer->>API: getWarpMessage(messageID)
    API->>State: Query message
    State-->>API: Return unsigned message
    API-->>Relayer: Unsigned message bytes
    
    Relayer->>Consensus: Request signatures (ACP-118)
    Consensus-->>Relayer: BLS signatures
    
    Relayer->>Relayer: Aggregate signatures
    Relayer->>CChain: Send signed message
    
    CChain-->>CChain:  Message received!
```

---

## Incoming Message Flow (C-Chain → WarpCustomVM)

```mermaid
sequenceDiagram
    participant CChain as C-Chain Contract
    participant CVal as C-Chain Validators
    participant Relayer as ICM Relayer
    participant API as API Server
    participant Builder
    participant Consensus as VM Validators
    participant Chain
    participant State as Database

    CChain->>CChain: sendWarpMessage(payload)
    CChain->>CVal: Create unsigned message
    CVal->>CVal: Sign message
    
    Relayer->>CVal: Monitor for Warp messages
    CVal-->>Relayer: Unsigned message detected
    
    Relayer->>CVal: Request signatures (ACP-118)
    CVal-->>Relayer: BLS signatures
    Relayer->>Relayer: Aggregate signatures
    
    Relayer->>API: receiveWarpMessage(signedMsg)
    API->>API: Parse signed message
    API->>API: Verify BLS signature
    API->>API: Extract AddressedCall payload
    
    API->>Builder: AddMessage(parsedMsg)
    API-->>Relayer: Return txId

    Note over Builder: Wait for block time (1 sec)
    
    Builder->>Builder: BuildBlock()
    Builder->>Builder: Include message
    Builder->>Consensus: Propose block to ALL validators
    
    Note over Consensus: Block propagates to all nodes
    
    Consensus->>Consensus: All validators vote
    Consensus->>Chain: ALL nodes call Accept(block)
    
    Note over Chain: Critical: Runs on EVERY validator
    
    Chain->>Chain: Detect external message<br/>(sourceChainID ≠ ourChainID)
    Chain->>State: SetReceivedMessage(msg)<br/>ON EVERY NODE
    
    Note over State: All validators now have<br/>identical message state
    
    State->>State:  Message synced across all nodes
```

---

## Consensus Synchronization

```mermaid
graph TD
    subgraph "Message Reception"
        MSG[Incoming Warp Message]
        API[API receives on Node 1]
        BUILDER[Builder queues message]
    end

    subgraph "Block Creation"
        PROPOSE[Node 1 proposes block<br/>with message]
        BROADCAST[Block broadcast to<br/>all validators]
    end

    subgraph "Consensus Voting"
        V1[Validator 1<br/>Vote: Accept]
        V2[Validator 2<br/>Vote: Accept]
        V3[Validator 3<br/>Vote: Accept]
        MAJORITY[Majority Reached]
    end

    subgraph "Synchronization (All Nodes)"
        A1[Node 1: Accept block]
        A2[Node 2: Accept block]
        A3[Node 3: Accept block]
        
        D1[Node 1: Detect external msg]
        D2[Node 2: Detect external msg]
        D3[Node 3: Detect external msg]
        
        S1[(Node 1 Database)]
        S2[(Node 2 Database)]
        S3[(Node 3 Database)]
    end

    subgraph "Result"
        SYNC[ All nodes have<br/>identical message state]
    end

    MSG --> API
    API --> BUILDER
    BUILDER --> PROPOSE
    PROPOSE --> BROADCAST
    
    BROADCAST --> V1
    BROADCAST --> V2
    BROADCAST --> V3
    
    V1 --> MAJORITY
    V2 --> MAJORITY
    V3 --> MAJORITY
    
    MAJORITY --> A1
    MAJORITY --> A2
    MAJORITY --> A3
    
    A1 --> D1
    A2 --> D2
    A3 --> D3
    
    D1 --> S1
    D2 --> S2
    D3 --> S3
    
    S1 --> SYNC
    S2 --> SYNC
    S3 --> SYNC

    style MSG fill:#fff4e1
    style SYNC fill:#c8e6c9
    style S1 fill:#e3f2fd
    style S2 fill:#e3f2fd
    style S3 fill:#e3f2fd
```

---

## Data Flow

```mermaid
flowchart LR
    subgraph "Input Sources"
        USER[User API Call]
        RELAYER[ICM Relayer]
    end

    subgraph "Processing Pipeline"
        PARSE[Parse & Validate]
        QUEUE[Message Queue]
        BLOCK[Block Builder]
        CONSENSUS[Consensus]
        ACCEPT[Accept Handler]
    end

    subgraph "State Storage"
        BLOCKS[(Block Headers)]
        MESSAGES[(Received Messages)]
        INDEX[(Message Index)]
    end

    subgraph "Query APIs"
        GET_MSG[getReceivedMessage]
        GET_ALL[getAllReceivedMessages]
        GET_BLOCK[getBlock/getLatestBlock]
    end

    USER -->|submitMessage| PARSE
    RELAYER -->|receiveWarpMessage| PARSE
    
    PARSE --> QUEUE
    QUEUE --> BLOCK
    BLOCK --> CONSENSUS
    CONSENSUS --> ACCEPT
    
    ACCEPT -->|Store Block| BLOCKS
    ACCEPT -->|Store External Msg| MESSAGES
    ACCEPT -->|Update Index| INDEX
    
    GET_MSG --> MESSAGES
    GET_ALL --> INDEX
    GET_ALL --> MESSAGES
    GET_BLOCK --> BLOCKS
    
    BLOCKS -.->|Message bytes| GET_BLOCK
    
    style USER fill:#e1f5ff
    style RELAYER fill:#fff4e1
    style BLOCKS fill:#e3f2fd
    style MESSAGES fill:#f3e5f5
    style INDEX fill:#fff3e0
```

---

## State Management

```mermaid
stateDiagram-v2
    [*] --> MessageCreated: User submits message
    
    MessageCreated --> Queued: Builder.AddMessage()
    Queued --> BlockProposed: Builder.BuildBlock()
    BlockProposed --> Voting: Validators receive block
    Voting --> Accepted: Majority vote
    Voting --> Rejected: Failed vote
    Rejected --> [*]
    
    Accepted --> Persisted: chain.Accept()
    
    state Persisted {
        [*] --> CheckSource
        CheckSource --> ExternalMessage: sourceChainID ≠ ourChainID
        CheckSource --> InternalMessage: sourceChainID = ourChainID
        
        ExternalMessage --> StoreReceived: SetReceivedMessage()
        InternalMessage --> StoreBlock: SetBlockHeader()
        
        StoreReceived --> [*]
        StoreBlock --> [*]
    }
    
    Persisted --> Queryable
    
    state Queryable {
        [*] --> APIs
        APIs --> GetBlock: getBlock()
        APIs --> GetMessage: getReceivedMessage()
        APIs --> GetAll: getAllReceivedMessages()
        
        GetBlock --> [*]
        GetMessage --> [*]
        GetAll --> [*]
    }
    
    Queryable --> [*]
```

---

## API Request Flow

```mermaid
flowchart TB
    subgraph "Client Request"
        CLIENT[HTTP Client]
        JSON[JSON-RPC Request]
    end

    subgraph "API Layer"
        ROUTER[RPC Router]
        AUTH[Authentication]
        HANDLER[Method Handler]
    end

    subgraph "Method Handlers"
        SUBMIT[submitMessage]
        RECEIVE[receiveWarpMessage]
        GETALL[getAllReceivedMessages]
        GETBLOCK[getBlock]
    end

    subgraph "Business Logic"
        CREATE[Create Warp Message]
        PARSE[Parse Signed Message]
        QUERY[Query Database]
    end

    subgraph "Backend"
        BUILDER[Builder Queue]
        STATE[State Reader]
    end

    CLIENT --> JSON
    JSON --> ROUTER
    ROUTER --> AUTH
    AUTH --> HANDLER
    
    HANDLER --> SUBMIT
    HANDLER --> RECEIVE
    HANDLER --> GETALL
    HANDLER --> GETBLOCK
    
    SUBMIT --> CREATE
    RECEIVE --> PARSE
    GETALL --> QUERY
    GETBLOCK --> QUERY
    
    CREATE --> BUILDER
    PARSE --> BUILDER
    QUERY --> STATE
    
    BUILDER -.->|Response| CLIENT
    STATE -.->|Response| CLIENT

    style CLIENT fill:#e1f5ff
    style BUILDER fill:#fff3e0
    style STATE fill:#e3f2fd
```

---

## Block Structure

```mermaid
graph TB
    subgraph "Block"
        HEADER[Block Header]
        
        subgraph "Header Fields"
            HASH[Block Hash]
            PARENT[Parent Hash]
            HEIGHT[Block Height]
            TIME[Timestamp]
            MSGIDS[Message IDs Array]
            WARPMAP[WarpMessages Map]
        end
        
        HEADER --> HASH
        HEADER --> PARENT
        HEADER --> HEIGHT
        HEADER --> TIME
        HEADER --> MSGIDS
        HEADER --> WARPMAP
    end

    subgraph "Message Storage"
        MSGIDS --> |msgID1| WARPMAP
        MSGIDS --> |msgID2| WARPMAP
        MSGIDS --> |msgID3| WARPMAP
        
        WARPMAP --> |"msgID1"| BYTES1[Unsigned Message Bytes]
        WARPMAP --> |"msgID2"| BYTES2[Unsigned Message Bytes]
        WARPMAP --> |"msgID3"| BYTES3[Unsigned Message Bytes]
    end

    subgraph "Unsigned Message Structure"
        BYTES1 --> UNWRAP1[NetworkID + SourceChain + AddressedCall]
        
        subgraph "AddressedCall"
            AC[CodecID + TypeID + SourceAddr + Payload]
        end
        
        UNWRAP1 --> AC
    end

    style HEADER fill:#e3f2fd
    style WARPMAP fill:#fff3e0
    style AC fill:#f3e5f5
```

---

## Database Schema

```mermaid
erDiagram
    BLOCKS ||--o{ MESSAGES : contains
    MESSAGES ||--|| MESSAGE_INDEX : indexed_by
    
    BLOCKS {
        bytes blockID PK
        bytes parentHash
        uint64 height UK
        int64 timestamp
        bytes[] messageIDs
        map warpMessages
    }
    
    MESSAGES {
        bytes messageID PK
        bytes sourceChainID
        bytes sourceAddress
        bytes payload
        int64 receivedAt
        uint64 blockHeight FK
        bytes signedMessage
        bytes unsignedMessage
    }
    
    MESSAGE_INDEX {
        string prefix
        bytes messageID FK
    }
```

---

## Warp Message Lifecycle

```mermaid
timeline
    title Cross-Chain Message Lifecycle
    
    section Creation
        Source Chain : Message created
        Source Chain : Wrapped in AddressedCall
        Source Chain : Included in block
    
    section Signature Collection
        Validators : Validators sign message
        ACP-118 : Relayer requests signatures
        Aggregation : BLS signatures aggregated
    
    section Delivery
        ICM Relayer : Delivers to destination
        Destination API : Parses signed message
        Destination API : Verifies signatures
    
    section Consensus
        Block Builder : Creates block with message
        Validators : Vote on block
        All Nodes : Accept block
    
    section Storage
        All Nodes : Detect external message
        All Nodes : Store in database
        All Nodes : Message queryable
```

---

## Error Handling Flow

```mermaid
flowchart TD
    START[Receive API Request]
    VALIDATE{Valid Request?}
    PARSE{Parse Success?}
    VERIFY{Verify Signature?}
    QUEUE{Queue Success?}
    CONSENSUS{Consensus Accept?}
    STORE{Store Success?}
    
    ERR_INVALID[Error: Invalid request]
    ERR_PARSE[Error: Parse failed]
    ERR_VERIFY[Error: Invalid signature]
    ERR_QUEUE[Error: Queue full]
    ERR_CONSENSUS[Error: Consensus failed]
    ERR_STORE[Error: Database error]
    
    SUCCESS[ Success]
    
    START --> VALIDATE
    VALIDATE -->|No| ERR_INVALID
    VALIDATE -->|Yes| PARSE
    
    PARSE -->|No| ERR_PARSE
    PARSE -->|Yes| VERIFY
    
    VERIFY -->|No| ERR_VERIFY
    VERIFY -->|Yes| QUEUE
    
    QUEUE -->|No| ERR_QUEUE
    QUEUE -->|Yes| CONSENSUS
    
    CONSENSUS -->|No| ERR_CONSENSUS
    CONSENSUS -->|Yes| STORE
    
    STORE -->|No| ERR_STORE
    STORE -->|Yes| SUCCESS
    
    ERR_INVALID --> |Log & Return| START
    ERR_PARSE --> |Log & Return| START
    ERR_VERIFY --> |Log & Return| START
    ERR_QUEUE --> |Retry| START
    ERR_CONSENSUS --> |Repropose| START
    ERR_STORE --> |Panic| START
    
    style SUCCESS fill:#c8e6c9
    style ERR_INVALID fill:#ffcdd2
    style ERR_PARSE fill:#ffcdd2
    style ERR_VERIFY fill:#ffcdd2
    style ERR_QUEUE fill:#fff9c4
    style ERR_CONSENSUS fill:#fff9c4
    style ERR_STORE fill:#ffcdd2
```

---

## Performance Metrics

```mermaid
graph LR
    subgraph "Timing Breakdown"
        A[API Request] -->|< 1ms| B[Parse Message]
        B -->|< 1ms| C[Queue Message]
        C -->|~1000ms| D[Block Creation]
        D -->|500-2000ms| E[Consensus]
        E -->|< 10ms| F[Database Write]
        F -->|< 1ms| G[Response]
    end

    subgraph "Total Latency"
        T1[Outgoing: 1.5-3 seconds]
        T2[Signature Collection: 2-5 seconds]
        T3[Incoming: 1.5-3 seconds]
        T4[End-to-End: 5-11 seconds]
    end

    style D fill:#fff9c4
    style E fill:#ffecb3
    style T4 fill:#c8e6c9
```

---

## How to View These Diagrams

### In GitHub/GitLab
Simply view this file - Mermaid diagrams render automatically.

### In VS Code
1. Install extension: "Markdown Preview Mermaid Support"
2. Open this file
3. Press `Ctrl+Shift+V` (or `Cmd+Shift+V` on Mac)

### Online Viewers
- [Mermaid Live Editor](https://mermaid.live/)
- Copy any diagram code and paste to edit/view

### Export as Images
Use Mermaid CLI:
```bash
npm install -g @mermaid-js/mermaid-cli
mmdc -i DIAGRAMS.md -o diagrams.pdf
```
