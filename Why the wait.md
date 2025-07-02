
# Why Fastlane Took Time

> Building a network benchmarking tool isn’t just about measuring speed — it’s about engineering precision.

Here’s what made Fastlane a beast to build:

* **No usable public TCP servers** — most speedtest tools rely on hosted infrastructure. I had to scout, validate, and hardcode servers that actually work.

* **Raw TCP? Not so fast.** Setting up real raw TCP tests across regions without running a global infra is a nightmare. So I dug into `iperf3`, learned how to use it programmatically, and made it work seamlessly inside a CLI.

* **Accuracy vs. Accessibility.** I refused to ship a half-baked tool that fakes speed via HTTP — so every bit transferred, every byte counted, had to reflect reality.

* **CLI polish takes time.** I didn’t want another boring terminal tool — I wanted Fastlane to feel like Warp, with real-time spinners, Unicode boxes, and nerd-mode flags. That meant building a clean UI layer from scratch.

* **I did it solo.** From networking internals to CLI design, concurrency, JSON reports, and even packaging — every bit was handcrafted.

So yeah... it took time. But I wasn't building just another tool —
**I was building the tool I always wished existed.**
