# nullrun design

What the game is, and the order to build it in. Each milestone runs on its own,
so there's always something playable.

## The idea

You're a netrunner. Each run you jack into a corporate network, steal some data,
and get out before the system traces you and fries your brain. The network is a
random graph of nodes. ICE are the enemies, data is the loot, dying is permanent.
A roguelike where the dungeon is a computer, which is why it's in a terminal.

## Three meters

**Integrity** is health. Zero means you flatline.

**Trace** is the clock. Everything you do fills it, and the system gets meaner as
it climbs. This is what forces "grab more data or bail?" every turn. It's the
most important thing to get right.

**RAM** is what programs cost to run, so you can't do everything at once.

## What it looks like

```
  ┌─ NETWORK: AZTECHNOLOGY-SUBNET-7 ──────────────────────┐
  │        [◊]────────[▓]          ??                      │
  │         │          │            :                      │
  │    [@]──┴──[·]─┤  [ICE] ├────┤  ??  │                  │
  │                └───┬───┘                               │
  │                   [$]  <- the data you're after        │
  ├────────────────────────────────────────────────────────┤
  │ INTEGRITY ████████░░  TRACE ███░░░░░░░  RAM ▓▓▓▓░       │
  └────────────────────────────────────────────────────────┘
```

`@` you, `◊` scanned node, `·` cleared, `▓`/`[ICE]` enemy, `$` objective,
`??` fog.

## v1

One network you can win or lose. Build in order, play after each step.

- [x] **M0.** gruid + tcell running, a map, `@` moving with hjkl/arrows, walls.
- [x] **M1.** Generate the node graph, draw it, fog so you only see what you scanned.
- [x] **M2.** Datastore node + exit at the entry. Grab data, get back out. Win condition.
- [ ] **M3.** Trace fills as you act, maxing it ends the run. Now it's actually a game.
- [ ] **M4.** One enemy: a sentry that damages integrity, and one program to break it.
- [ ] **M5.** A small program kit (icebreaker, scanner, cloak) gated by RAM.

M5 done = a real, self-contained game worth shipping.

## v2

- More networks, rising security. Your dungeon levels.
- More ICE types: barriers, trace spikers, lethal black ICE.
- Keep credits between runs, spend on programs and cyberware. Biggest lever for
  making people keep playing.
- Cyberware that trades power for humanity/max integrity.
- Stolen data as readable flavor (emails, memos, secrets).

## v3

- Daily seed: everyone plays the same network, compares scores.
- Meatspace: walk the city between runs, take jobs, then jack in. Long-term goal.

## Tech

Go (single static binary, easy to hand out).
[gruid](https://codeberg.org/anaseto/gruid) for the grid: model/update/draw loop,
tcell driver, `rl` package (FOV, map gen, turn-order event queue), `paths` for
A*/Dijkstra/JPS when ICE needs to chase you.

## Don't forget

- Every milestone runs.
- Trace is the soul. When it feels off, it's usually the trace tuning.
- The terminal is the point.
