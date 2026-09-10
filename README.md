# nullrun

A cyberpunk hacking roguelike for your terminal.

Jack into a corporate network, grab the data, and get out before the system
traces you and flatlines your brain. Every run is a fresh procedurally generated
intrusion. Death is permanent.

```
  ┌─ NETWORK: AZTECHNOLOGY-SUBNET-7 ──────────────────────┐
  │        [◊]────────[▓]          ??                      │
  │         │          │            :                      │
  │    [@]──┴──[·]─┤  [ICE] ├────┤  ??  │                  │
  │                └───┬───┘                               │
  │                   [$]  <- the data you're after        │
  ├────────────────────────────────────────────────────────┤
  │ INTEGRITY ████████░░  TRACE ███░░░░░░░  RAM ▓▓▓▓░       │
  │ > A sentry ICE blocks the way south. It hasn't seen    │
  │   you yet.                                              │
  └────────────────────────────────────────────────────────┘
```

Early days. Still building the first playable slice. See [`DESIGN.md`](DESIGN.md)
for the plan.

## Install

Needs [Go](https://go.dev/dl/) 1.21+.

```sh
go install github.com/artifactNU/nullrun@latest
```

Or from source:

```sh
git clone https://github.com/artifactNU/nullrun.git
cd nullrun && go build -o nullrun . && ./nullrun
```

## Playing

Reach the datastore (`$`), crack it, and get back to where you jacked in before
the trace meter fills. Watch three things:

- **Integrity** is your health. Hit zero and you flatline. That's permadeath.
- **Trace** is the clock. Everything you do fills it, and the system gets nastier as it climbs.
- **RAM** is what your programs run on. Spend it wisely.

Move with `hjkl` or arrows, `s` to scan, `b` to break ICE, `c` to cloak,
`q` to jack out. `?` for help.

`@` is you, `$` the data, `▓`/`[ICE]` the things trying to stop you, `??` fog.

## Built with

[gruid](https://codeberg.org/anaseto/gruid) for the grid, FOV, pathfinding, and map generation.

## License

GPLv3. See [LICENSE](LICENSE).
