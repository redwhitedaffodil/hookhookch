## Usage
- Download Go (https://go.dev/dl)
- Compile with `go build .`
- See the help menu with `./chesshook2`
- Set the keys in `config.json`. This will be automatically created if you try running something.
- Go write strategies in `strategies.json`. The format should be pretty self explanatory. The keys are documented in `config.go`.
- The software is still in development so you might have to go into `db.json` manually sometimes. Try not to mess it up too much.

## Features
- Multiple accounts running concurrently
- Support for different solving strategies for each account
  - Can aim for a target solve count or a target rating
- Logging to a discord webhook (you might run this as a cron job)
- Remembers when the a free account was run last to stay withing the free limit of 3

![CLI](.github/img/demo.png)

You can get absurdly high training time using the "hour" strategy:

![Many hours](.github/img/spoofed-time.png)

## Getting tokens
Tokens don't seem to expire for some time, but I have yet to check long-term. I've had some tokens expire fast and some not.
It probably has something to do with the cookie that you copy, try to get one that has as few parameters and avoid anything cloudflare: `"cf"`

1. Open devtools (generally f12), the network tab and filter by Fetch/XHR

![Devtools](.github/img/getting-a-token-1.png)

2. Find any "authenticated" looking request, examples include `notices`, `QueryFriends`, `DispatchEventBatch`, `{YOURUSERNAME}`, I find `notices` to be fairly frequent. Copy it as cURL.

![Network requests](.github/img/getting-a-token-2.png)

3. Execute `./chesshook2 accounts add` and follow the instructions to paste it in. It will parse your data.

## Known issues
Sometimes it does these:

![Weird submission](.github/img/incomplete-solution.png)
![Wrong submission](.github/img/wrong-solution.png)

It shouldn't matter though, these are quite infrequent and I think they're associated with unexpected stopping and starting.