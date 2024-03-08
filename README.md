# Signin Website for bush
- Uses ephemeral URLs with new UUIDs generated each time a form is submitted
- QR codes can be displayed from a password protected web page, using websockets to update the QR image
- Written in go for memory saftey, great concurrency, and speed

## Secure and reliable
- More reliable and secure than pen and paper.
- Uses google signin for getting name and email
  - You can limit sign ins to certain accounts or orgs.
- Form data submitted is thouroughly validated and sanitized.
- Can be hosted on a local network to prevent access to the website from outside of wifi


## Features to consider
- Using google authentication and possibly requiring you to be signed into a special google account to access websocket/qr viewer.
