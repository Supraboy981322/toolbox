# toolbox

simple http server with basic tools.

---

## features
>[!NOTE]
>this list isn't guaranteed to be up-to-date
>  each feature has it's own function, (which doesn't call another function)
>    so just read the function names if you want a more up-to-date list

- random string/password generator
  - (basic) password builder build tool
- url de-shortener
- [NaaS](https://github.com/hotheadhacker/no-as-a-service) (returns `text/plain` instead of JSON)
- Send a message to Discord (via a webhook)
- A web server (with BHTM, Bash in HTML)

<sub>Instead of packaging the whole <a href="https://github.com/Supraboy981322/ELH">ELH</a> server and having to mirror updates between the two repos, I created BHTM, which is just a stripped-down version of ELH that only uses Bash. Why use Bash, it's available on almost every Linux system, and if I'd like this server to run on any Linux machine, while still using something otherthan JS/TS, PHP, or plain HTML, Bash seemed like the easiest option for that purpose. (The Bash libs for BHTM are ahead of ELH, at the moment, but I plan to fix that at some point)</sub>
