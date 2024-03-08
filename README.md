# Signin Website for bush
- Uses ephemeral URLs with new UUIDs generated each time a form is submitted
- QR codes can be displayed from a password protected web page, using websockets to update the QR image
- Written in go for memory saftey, great concurrency, and speed

## Secure and reliable
- More reliable and secure than pen and paper.
- Uses google signin for getting name and email
  - You can limit sign ins to certain accounts or orgs.
- Form data submitted is thouroughly validated and sanitized.
- Can be hosted on a local network to prevent access to the website when trying to connect outside of bush wifi


## Things that still need to be figured out (not hard to solve)
- Using google authentication and possibly requiring you to be signed into a special google account to access websocket/qr viewer.
- Need more info on how we want to display the actual QR code, but the backend is mostly solved
- Deploying with NGINX
