# ttyper

A terminal typing speed test. Requires Node.js - no npm packages.

![demo](https://i.imgur.com/placeholder.png)

## Usage

```sh
node ttyper.js              # 25 common words (default)
node ttyper.js -n 50        # 50 words
node ttyper.js -t 30        # 30-second timed mode
node ttyper.js -m code      # programming vocabulary
node ttyper.js -m quotes    # famous quotes
node ttyper.js -h           # show help
```

## Controls

| Key      | Action                                  |
|----------|-----------------------------------------|
| `type`   | Letters fill in as you type             |
| `space`  | Submit current word                     |
| `⌫`      | Delete character - hold to go back through words |
| `tab`    | Restart                                 |
| `ctrl+o` | Open settings menu                      |
| `ctrl+c` | Quit                                    |

## Modes

- **common** - top ~200 most frequent English words
- **code** - programming keywords and terms (JS/TS focused)
- **quotes** - famous quotes typed as full phrases, cycling through all quotes before repeating

## Settings menu

Open with `ctrl+o`. Navigate with `↑↓` arrow keys and press `enter` to select.

Settings are saved to `~/.ttyper.json` and restored on next launch.

| Setting    | Options               |
|------------|-----------------------|
| Mode       | common / code / quotes |
| Word count | 10 / 25 / 50 / 100    |
| Time limit | off / 15s / 30s / 60s |

Switching to a time limit overrides word count mode, and vice versa.

## Metrics

- **WPM** - calculated as total characters typed / 5 / elapsed minutes (industry standard)
- **Accuracy** - correct keystrokes / total keystrokes, tracked per character in real time
