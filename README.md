# COMP 8005 - Assignment 4

## Structure

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

## Running

```bash
./build.sh

./build/controller ...args
./build/worker ...args
```

Run `chmod +x` on the build script if there is a permission issue with
executing it.
