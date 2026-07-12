# Quickstart: the `soulstream` CLI

```sh
go build -o bin/soulstream ./cmd/soulstream

# Configure once (a NATS context supplies the server + creds):
nats context add soulstream --server nats://127.0.0.1:4222
export SOULSTREAM_CONTEXT=soulstream SOULSTREAM_REALM=acme SOULSTREAM_PERSONA=daan

# Set up and look around:
soulstream provision
soulstream board

# Start a topic and converse:
path=$(soulstream start "Q2 VAT filing" --subject "the Q2 return" --tag finance)
soulstream post "$path" "numbers are in, @bookkeeper-agent please check box 5"
soulstream show "$path"
soulstream show "$path" --json          # machine-readable

# Files:
soulstream attach "$path" ./q2-lines.csv --type text/csv   # prints the object key
soulstream get "attachments/$path/<id>" ./out.csv          # verifies the digest

# Live:
soulstream watch "$path"      # streams updates until Ctrl-C
soulstream inbox              # streams your @mentions until Ctrl-C

# Wrap up:
soulstream close "$path"
```

## Verify (definition of done)

```sh
make check   # fmt + tidy + build + test + lint, all green, none skipped
```

Command logic is tested against an in-process JetStream server via the injectable `Run(...)`; the
`cmd/soulstream` binary is a two-line wrapper.
