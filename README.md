# blindbot

Blindbot is a Slack bot to play a participative blindtest. Players submit challenges and play, blindbot do the rest.

## Use it
```slack
/submit https://www.youtube.com/watch?v=oHg5SJYRHA0 "reponse 1,reponse 2" "hints"
```

After submitting a Youtube link, blindbot will convert it into an MP3 sample hosted on your server. Then the bot will post the challenge in the channel `blindtest` with an anonymised link. 

When a player gives the right answer in the challenge's thread the bot add an emoji to let other players know this challenge is completed.

## Incoming features
* Statistics (best player, best submitter, per day, per week).
* https (almost done)
* Spotify integration
* Clean up files after a timeout (e.g. 2 weeks)
* Draw an icon for blindbot
* Web view for statistics.

## Run it
Configure the following variables in [update-and-deploy.sh](scripts/update-and-deploy.sh):
```bash
DOMAIN_NAME="example.org" # your server domain name
SLACK_KEY="XXXXX..." # Bot User OAuth Access Token
SLACK_OAUTH2_KEY="XXXXXXXXXX..." # OAuth Access Token
SLACK_MASTER="master.email@domain.com" # your email (gives logging and advanced command in Slack)
```

Then run it:
```bash
./update-and-deploy.sh

# or a particular version
VERSION=v0.3.0 ./update-and-deploy.sh
```

## Notes
The following folders will be created:
* `db/` - the database.
* `music/` - where musics are stored.
* `cred/` - https certificate.