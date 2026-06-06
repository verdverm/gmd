# launching agent harnesses from gmd

We want to be able to start a user's preferred harness from the gdm cli, prepopulated with message and/or other preperation (flags, custom config file path for "this" run)
- different ways to do this based on harness, let's just use exec
- do opencode for now, prepare for expansion to other cli, there should be config for these as well so user can reference by name

Do the following:
1. a './pkg/agent/' with the abstraction and specifics
2. a 'gdm agent' command which can launch various agent harnesses like opencode and claude.
3. update my global config for the new content