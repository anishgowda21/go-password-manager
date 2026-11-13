# go-password-manager

A small zero-knowledge password manager built with Go and React, made as a learning project to see how this kind of app works.

Live demo: https://gopmpr.cloud.run/

> **Experimental demo, not a production app.** It has not been audited, and the SQLite database lives on the server's local disk, so on ephemeral hosting (e.g. Cloud Run) your data can be wiped at any time. Don't store real passwords.

# What is a zero-knowledge password manager

In simple words, the server will have zero knowledge regarding your stored passwords. What the server sees is just an encrypted blob along with an encrypted vault key, which must be decrypted to be used. Every encryption and decryption operation happens locally on the client side, and the server only knows about the final encrypted blob and your encrypted key. So even in the case of a stolen or leaked database, the attacker will have a (very) hard time getting hold of your actual unencrypted data.

# How it all works

## 1. Signup: deriving keys from the master password

It all starts with user signup. When the user signs up and enters their master password, the client locally generates a 16-byte random salt. This, along with some default settings, is sent to be hashed by an algorithm called `Argon2id`, which produces a 32-byte hash. The hash produced by this algorithm is deterministic for the same inputs, which is the reason the settings and salt used by the user are stored on the server for future authentication. (Note: the same hash cannot be generated without the user's actual master password, which the server never sees.)

```
┌─────────────────┐
│ Master password │──┐
└─────────────────┘  │
┌─────────────────┐  │   ┌───────────┐   ┌────────────────────┐
│ 16-byte salt    │──┼──►│ Argon2id  │──►│ 32-byte master key │
└─────────────────┘  │   └───────────┘   └────────────────────┘
┌─────────────────┐  │
│ KDF settings    │──┘
└─────────────────┘
```

Once the 32-byte hash is derived from the user's password and the salt, the same hash derived from Argon2id is used to generate two more one-way hashes: the `encryption key` and the `auth key`.

- `encryption key`: This is the main key which is used to encrypt or decrypt your vault key.
- `auth key`: This is sent to the server and hashed again inside the server, and only its hash is stored. This is used to authenticate the user on login.

```
┌─────────────────────────────────────────────┐
│ 32-byte master key (Argon2id output)        │
└──────────────────────┬──────────────────────┘
                       │ HKDF-Expand
           ┌───────────┴───────────┐
           ▼                       ▼
┌─────────────────────┐ ┌─────────────────────┐
│ Encryption key      │ │ Auth key            │
│ (wraps vault key)   │ │ (sent to server)    │
└─────────────────────┘ └─────────────────────┘
```

## 2. Vault key: creating and encrypting it

Once these two are generated, the client then generates a random 32-byte `vault key`. This is kind of the whole key to your storage. This is the main key which can encrypt and decrypt your login items. The most important thing here is that this key, in its decrypted form, must not be stored anywhere, even in session storage. Always keep the decrypted version in the client's memory only.

This generated `vault key` is then encrypted by our previously generated `encryption key`. The encryption algorithm used here is `AES-GCM`. The GCM algorithm needs a random nonce to be in front of the blob, along with a version indicator (to keep track of which encryption algorithm is used). It also takes AAD (additional authenticated data). It's not stored in the blob; here it's a context string used to produce the auth tag. The auth tag is a short checksum that GCM computes itself during encryption, and it uses the ciphertext and the AAD to calculate it. When decrypting, we need to pass the same AAD, and GCM will make sure of two things:

1) the ciphertext is intact
2) it is being deciphered with the correct context.

Overall, the final output after the encryption looks like `[version][nonce][ciphertext][auth tag]`.

```
┌──────────────────────┐
│ Vault key (32 bytes) │──┐
└──────────────────────┘  │
┌──────────────────────┐  │   ┌──────────────┐   ┌─────────────────────┐
│ Encryption key       │──┼──►│ AES-256-GCM  │──►│ Encrypted vault key │
└──────────────────────┘  │   └──────────────┘   └─────────────────────┘
┌──────────────────────┐  │
│ AAD (context string) │──┘
└──────────────────────┘

Encrypted vault key, as stored:
┌─────────┬──────────┬────────────┬──────────┐
│ version │  nonce   │ ciphertext │ auth tag │
│ 1 byte  │ 12 bytes │            │ 16 bytes │
└─────────┴──────────┴────────────┴──────────┘
```

## 3. What the server receives at signup

After this we are left with an encrypted `vault key`. Once all this is finished on the client side, we will send the items below to the server:

```
1) email
2) encrypted vault key
3) salt
4) KDF settings
5) auth key
6) device-id
7) device-name
```

So, essentially these are the items related to a user that are stored in the database, along with the encrypted blob items we will send later. This is the real benefit of the zero-knowledge architecture here: even if any of these items leak, none of the real data of a user, that is the master password or the stored items, can be retrieved, unless the user had a weak master password that can be derived using brute force. If the master password is strong, even the leaked data cannot produce a usable value for the attacker.

```
┌──────────────────────────┐        ┌──────────────────────────┐
│ Client sends             │        │ Server stores            │
├──────────────────────────┤        ├──────────────────────────┤
│ email                    │        │ email                    │
│ encrypted vault key      │        │ encrypted vault key      │
│ salt                     │ ─────► │ salt                     │
│ KDF settings             │        │ KDF settings             │
│ auth key                 │        │ hash of auth key         │
│ device-id, device-name   │        │ device-id, device-name   │
└──────────────────────────┘        └──────────────────────────┘

Never sent: master password, encryption key, unencrypted vault key
```

## 4. Storing login items

Essentially, the same process is repeated for encrypting the login items. First, the user fills in the form with the login credentials they want to store. That form data is converted to JSON, and a unique UUID is generated on the client for each record. The JSON is then encrypted with the vault key using AES-GCM, with the UUID passed in as the AAD (additional authenticated data), which ties the ciphertext to that specific item. This produces the encrypted blob.

Each item also has a revision number, which keeps the client and server in sync so that edits don't silently overwrite each other. A new entry starts at revision 1, and every time it's edited the server increments it. When the client saves an edit, it sends the revision it last saw, and the server only accepts the update if that revision still matches; otherwise it reports a conflict.

So the final thing we end up with is an ID, a revision number and the encrypted blob, and that is what gets stored on the server. Again, the server doesn't know any of the sensitive info the user stores. Everything is encrypted and can only be decrypted with the vault key, which is held only in the client's memory while the user is using the application.

```
┌─────────────────┐
│ Item JSON       │──┐
└─────────────────┘  │
┌─────────────────┐  │   ┌──────────────┐   ┌────────────────┐
│ Vault key       │──┼──►│ AES-256-GCM  │──►│ Encrypted blob │
└─────────────────┘  │   └──────────────┘   └────────────────┘
┌─────────────────┐  │
│ AAD = item UUID │──┘
└─────────────────┘

┌───────────────────────────────────────────────────┐
│ Create   : send ID + blob, server sets revision 1 │
│ Edit     : send last-seen revision + blob         │
│            server increments it if it matches     │
│ Mismatch : server rejects it (conflict)           │
└───────────────────────────────────────────────────┘
```

## 5. Logging in

Login works mostly like signup. The user enters their email and master password. Before anything else, the client sends a pre-login request to the server with the email. The server checks whether an account exists for that email. If it does, the server sends back the salt and KDF settings that the client generated and the server stored during the user's original signup.

If the account doesn't exist, the server instead generates a fake 16-byte salt and sends it back, along with default KDF settings. This is so that someone can't easily guess whether an account exists for a given email. It isn't a perfect flow, since the same information can still be revealed when someone tries to sign up with that email, but it is a standard approach.

Once the client receives the KDF settings and salt, it repeats the same process we described before: it uses the master password to derive the encryption key and the auth key, and then sends the auth key to the server. The server hashes the received auth key and compares it with the hash it stored for that email. If they match, the server issues the credentials needed for syncing and viewing the items. If not, the request is rejected.

As mentioned before, Argon2id is deterministic. The same master password with the same KDF settings and the same salt always produces the same hash, and therefore the same encryption key and auth key. That is what proves the same user is logging in with the correct master password.

```
      Client                                         Server
         │                                              │
         │─ 1. pre-login: email ───────────────────────►│  account exists?
         │◄──────────────────── 2. salt + KDF settings ─│  (fake salt if no account)
         │   client derives encryption key + auth key   │
         │─ 3. login: auth key ────────────────────────►│  hash(auth key) = stored hash?
         │◄─────────── 4. tokens + encrypted vault key ─│  (rejected if no match)
         │         client unwraps the vault key         │
         │                                              │
```

## 6. Sessions and tokens

For credentials I'm using basic web authentication: a short-lived JWT access token and a longer-lived refresh token, which is stored (as a hash) in the sessions table.

For every protected request, such as getting, adding, updating or deleting login items, the client sends the access token as a Bearer token. The server checks it, and if it's valid we let the request through, otherwise it's rejected.

The refresh token is used when the access token expires. The client sends the refresh token to get a new access token. The server also replaces the refresh token in the sessions table with a new one, so the old refresh token can no longer be used, and sends the new tokens back.

During login we also send a device ID. Each user gets one session per device, so if the same user logs in again from the same device, the previous session is replaced and its old credentials stop working. This is a standard process used all over the web.

```
      Client                                         Server
         │                                              │
         │─ 1. request + access token ─────────────────►│  token valid? then process
         │◄──────── 2. response (401 if token expired) ─│
         │─ 3. refresh token ──────────────────────────►│  hash matches session? rotate
         │◄──────────── 4. new access + refresh tokens ─│  (old refresh token is dead)
         │─ 5. retry request + new access token ───────►│
         │◄─────────────────────────────── 6. response ─│
         │                                              │
```

# Tech Stack

```
Frontend   React, TypeScript, Vite, Tailwind CSS, zustand
Backend    Go, SQLite
Crypto     Go compiled to WebAssembly (Argon2id, HKDF-SHA256, AES-256-GCM)
Auth       JWT access tokens + rotating refresh tokens
Deploy     Docker, Google Cloud Run
```

# Architecture

The frontend is a regular React app, and the backend is a Go server that stores everything in SQLite. The same Go server also serves the built frontend.

All cryptographic operations (key derivation, key wrapping, and item encryption and decryption) are written in Go and compiled to WebAssembly, and they run in the browser. Argon2id is deliberately expensive, so the WASM module runs inside a Web Worker. This keeps the UI responsive while the keys are being derived.

```
Browser                                       Server
┌──────────────────────────────┐             ┌──────────────────┐
│ React UI                     │             │ Go API           │
│   │                          │  encrypted  │   │              │
│   ▼                          │    data     │   ▼              │
│ Web Worker (Go → WASM)       │ ──────────► │ SQLite           │
│   key derivation + encrypt   │             │ (blobs, hashes)  │
│   / decrypt                  │ ◄────────── │                  │
│                              │             │                  │
│ Master password and vault    │             │ Never sees the   │
│ key stay here, in memory     │             │ plaintext        │
└──────────────────────────────┘             └──────────────────┘
```
# Running it

The server needs two secrets, each at least 32 characters. Generate them once and keep reusing the same values. If `AUTH_HASH_KEY` changes, existing accounts can no longer log in.

```sh
export JWT_SECRET="$(openssl rand -base64 48)"
export AUTH_HASH_KEY="$(openssl rand -base64 48)"
```

The app runs on port `8090`. Open http://localhost:8090 after starting it.

## Option 1: Pre-built Docker image

```sh
docker run -p 8090:8090 -e JWT_SECRET -e AUTH_HASH_KEY \
  ghcr.io/anishgowda21/go-password-manager:latest
```

## Option 2: Build the Docker image yourself

```sh
git clone --depth 1 https://github.com/anishgowda21/go-password-manager
cd go-password-manager
docker build -t go-password-manager .
docker run -p 8090:8090 -e JWT_SECRET -e AUTH_HASH_KEY go-password-manager
```

## Option 3: Run locally with Make

Requirements: Git, Go and [Bun](https://bun.sh).

```sh
git clone --depth 1 https://github.com/anishgowda21/go-password-manager
cd go-password-manager
make run
```

`make run` installs the client dependencies, builds the WASM crypto module, builds the client, and starts the server. You can also run the steps on their own:

```
make install-client   # install client dependencies
make wasm             # build crypto.wasm into client/public
make build-web        # build the client and copy it into the server
make run              # build everything and start the server
```

The data is stored in a local SQLite file, so with Docker it is lost when the container is removed.

# License

This project is licensed under the MIT License.
See the [LICENSE](LICENSE) file for details.