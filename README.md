# bbs-go

A tiny TCP server implementing a basic Buletin Board Service.

```shell
telnet localhost 2323
```

```text
+-------------------------------------------+
|       __    __                            |
|      / /_  / /_  _____      ____  ____    |
|     / __ \/ __ \/ ___/_____/ __ \/ __ \   |
|    / /_/ / /_/ (__  )_____/ /_/ / /_/ /   |
|   /_.___/_.___/____/      \__, /\____/    |
|                          /____/           |
|                                           |
+-------------------------------------------+
vxn-dev bbs-go service (0.6.1)
telnet localhost 5555

> test
*** Invalid command

> exit
*** Bye
```

## build and run

```shell
# Copy example dotenv file
cp .env.example .env

# Edit some vars
vim .env

# Apply settings, build the binary and run it with env vars exported
make run
```

## system structure (TODO)

+ message boards
+ TUI forums
+ online games
+ user administration
