# Headless WM

This is a purely headless, API-driven, X11 window manager. It is meant
for kiosks, public displays, and other situations where interaction
with the screen / machine is limited (for users and/or staff), and
some form of remote control over the display is necessary.

It has no keybindings, no window decorations, no workspaces; it's
almost completely bare-bones except for keeping a list of
active/managed clients which can be manipulated through a (TODO) HTTP
API.

## Lineage

This is a fork of [rollcat's `dewm`](https://github.com/rollcat/dewm),
which is a fork of [Dave MacFarlane's `dewm`](https://github.com/driusan/dewm),
which includes bits from [taowm](https://github.com/nigeltao/taowm).
