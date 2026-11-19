# Auction

## Starting the system

Run all commands from the root of the repository.

Start each server node in its own terminal. Start all three within 5 seconds of each other.

**Terminal 1**
```bash
go run ./Server -id 1
```

**Terminal 2**
```bash
go run ./Server -id 2
```

**Terminal 3**
```bash
go run ./Server -id 3
```

Start a client in a fourth terminal:

**Terminal 4**
```bash
go run ./Client -name alice
```

Commands in the client:

```
bid <amount>
result
```

The auction runs for 100 seconds from when the servers are started.

## Testing a crash

Press `Ctrl+C` in the terminal of the current leader (node 1). After a few seconds another node takes over as leader, and bids can still be placed.

## Stopping the system

Press `Ctrl+C` in each terminal.