# Eve Discord Killmail Webhook

This is a simple script to query for killmails by
character, corporation, or alliance names/IDs, and echo
the killmails to Discord (or just to your console) as
they happen.

    fsufitchi @ ~ → evediscordkm
      -all string
        	comma-separated list of alliance names or IDs
      -char string
        	comma-separated list of character names or IDs
      -corp string
        	comma-separated list of corporation names or IDs
      -discord string
        	Discord webhook URL

    fsufitchi @ ~ → evediscordkm -corp "Tribal Liberation Force,24th Imperial Crusade,Federal Defense Union,State Protectorate"
    **jo Hamu** was killed (*Capsule*; *10,000* ISK) by 7 attacker(s) in **XHQ-7V** -- https://zkillboard.com/kill/58436557
    **Servamp Anthar** was killed (*Capsule*; *10,000* ISK) by 1 attacker(s) in **Kourmonen** -- https://zkillboard.com/kill/58435664
    **Mamotromico Cyberian** and 4 other(s) killed **Heigar Omaristos** (*Algos*; *22,677,682.75 ISK*) in **Oicx** -- https://zkillboard.com/kill/58436085
    **Mamotromico Cyberian**, Chomitin Bane and 6 other(s) killed **Timoti Rajas** (*Algos*; *20,911,491.02 ISK*) in **Oicx** -- https://zkillboard.com/kill/58436093

##Downloads

Go to the [Releases Page](https://github.com/fsufitch/evediscordkm/releases) to find the latest version for your OS. If your OS is not there, set up a Go development environment and run:

    go build github.com/fsufitch/evediscordkm
