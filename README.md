# COMP 8005 - Assignment 1

# Structure

```bash
cmd/                  # individual programs
  controller/         
  worker/
data/                 # sample/test data for programs
internal/             # lib
  controller/         # controller-specific libs
  shared/             # common libs
  worker/             # worker-specific libs
```

# Running

```
go build controller/main.go
go build worker/main.go
```

or 

```
go run controller/main.go ...args
go run woker/main.go ...args
```
