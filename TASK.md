# launching agent harnesses from gmd

Answers to follow up questions:

1. yes
2. yes, this should also work with 1, args: <name> "<message>"
3. from PATH
4. yes, so we shouldn't need the --launch-agent (that is default w/o flags), we should have have an --async flag and just not attach
5. exactly
6. yes, good idea
7. sure
8. no, let's refactor now, the harness list should be consistent throughout and use config-driven only
9. it should fail, we should have session command with management subcommands
10. yes, auto add if not in the file
11. prefer the checked out branch
12. we should have lifecycle/management commands, see also / pair with (9)
13. yes, rollback, another place we see errors more often here is in launching the agent, hence the two temp files. There can be backticks and such in the first message bound for the agent harness