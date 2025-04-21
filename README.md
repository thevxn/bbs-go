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
telnet localhost 2323

Enter username (or type 'register'): register
Choose a username: noarche
Choose a password:
Registration successful.
> help
*** Commands:
    help           --- show this help message
    post <message> --- post a message to the board
    read           --- read recent messages
    exit           --- quit the session

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
