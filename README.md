# Bush Campus Signin
a simple webapp written in go to allow students at bush to sign in and out of campus.

## basic rundown

1. A student visits the URL through the QR code (if the app is launched with a QR password)
2. The student then signs in with their google account if they have not already to verify their identity
3. The student then submits the form, saying whether or not they are signing in/out and specifying the reason
4. This data is then uploaded to a google sheet where the school can monitor who is on campus and who is not

## How do I configure/run this application

This application is deployed using containers, and in the future will use nginx as a reverse proxy (currently uses caddy)
To run this application you must properly configure the google console and put in the correct environment variables

1. create a new directory to store application data
1. copy the [.env template](.env.template) to a `.env` file for all of the credentials required to run the application
1. copy the sample [docker-compose.yml](docker-compose.yml) and modify it to your needs
1. create an empty directory `data`, this will be mounted onto the container and hold encrypted session data
1. create an [oauth 2.0 webapp in google cloud console](https://support.google.com/cloud/answer/6158849?hl=en)
    - in the `.env` file you copied, set the `GOOGLE_OAUTH_CLIENT_ID` and `GOOGLE_OAUTH_CLIENT_SECRET` appropriately from the JSON values downloaded
1. create a [service account through the google cloud console](https://cloud.google.com/iam/docs/service-accounts-create)
    - be sure to save the private key into the `.env` file under `GOOGLE_SREADSHEET_ACCOUNT_KEY` and the email under `GOOGLE_SREADSHEET_ACCOUNT_EMAIL`
1. Next, create a spreadsheet for storing the attendence data, make sure to share it with the service account you created earlier, inputting the email assigned to it.
    - copy the spreadsheet ID from the url and put it inside of the `.env` file in the field `GOOGLE_SPREADSHEET_ID`
1. using the sample [docker-compose.yml](docker-compose.yml) you are now (hopefully) able to run:
  ```
  docker-compose -d up
  ```

## FAQ
Q: why not use microsoft OAuth
A: their APIs are more challenging to work with, google simply makes it easier for developers to use their tooling, along with their better documentation. You can do all of the same things you can do with the google account that you can with the microsoft for this usecase so it really does not matter.

Q: why does this exist?
A: Keeping track of attendence on a paper sheet to know who is on campus and who is not, can be a hard and error prone task, by automating this, it saves significant time and energy

Q: when will this be deployed?
A: hopefully before summer.
