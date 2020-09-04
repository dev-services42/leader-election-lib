# Simple leader election

Run 3 consul agent on local ports: `127.0.0.1:8500`, `127.0.0.1:8501`, `127.0.0.1:8502`.

Then run 3 example processes in separate terminal sessions:

```bash
# Terminal 1
export CONSUL_ADDR=127.0.0.1:8500
go run main.go

# Terminal 2
export CONSUL_ADDR=127.0.0.1:8501
go run main.go

# Terminal 3
export CONSUL_ADDR=127.0.0.1:8502
go run main.go
```

Try to:
- kill terminal sessions
- release lock sessions on consul agents
- remove kv key from consul KV
