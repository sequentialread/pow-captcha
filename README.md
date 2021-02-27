# 💥PoW! Captcha

A proof of work based captcha similar to [friendly captcha](https://github.com/FriendlyCaptcha/friendly-challenge), but lightweight, self-hosted and GPLv3 licensed. All dependencies are included, total front-end unminified gzipped file size is about 50KB.

![screencast](readme/screencast.gif)


### Table of Contents

 1. [How it works](#how-it-works)
 1. [What is Proof of Work?](#what-is-proof-of-work)
 1. [Overview sequence diagram](#overview-sequence-diagram)
 1. [Configuring](#configuring)
 1. [HTTP Captcha API](#http-captcha-api)
 1. [HTTP Admin API](#http-admin-api)
 1. [HTML DOM API](#html-dom-api)
 1. [Running the example app](#running-the-example-app)
 1. [Implementation walkthrough via example app](#implementation-walkthrough-via-example-app)
 1. [Implementation Details for Developers](#implementation-details-for-developers)
 1. [What is Proof of Work? Extended Concrete Example](#what-is-proof-of-work-extended-concrete-example)

# How it works 

This application was designed to be a drop-in replacement for ReCaptcha by Google. It works pretty much the same way;

 1. Your web application requests a captcha (in this case, a batch of captchas) from the captcha HTTP API
 2. Your web application displays an HTML page which includes a form, and passes the captcha data to the form
 3. The HTML page includes the JavaScript part of the Captcha app, this JavaScript draws the Captcha on the page  
 4. When the Captcha is complete, its JavaScript will fire off a callback to your JavaScript (usually used to enable the submit button on the form)
 5. When the form is submitted, your web application submites the captcha result to the captcha HTTP API for validation

# What is Proof of Work? 

Proof of Work (PoW) is a scheme by which one computer can prove to another that it expended a certain amount of computational effort. 

PoW does not require any 3rd party or authority to enforce rules, it is based on mathematics and the nature of the universe.

PoW works fairly well as a deterrent against spam, a PoW requirement makes sending high-volume spam computationally expensive.

It is impossible to predict how long a given Proof of Work will take to calculate. It could take no time at all (got it on the first try 😎 ), or it could take an abnormally long time (got unlucky and took forever to find the right hash 😟 ). You can think of it like flipping coins until you get a certain # of heads in a row.  This **DOES** matter in terms of user interface and usability, so you will want to make sure that the difficulty is low enough that users are extremely unlikely to be turned away by an unlucky "takes forever" captcha.

The word ["Nonce"](https://en.wikipedia.org/wiki/Cryptographic_nonce#Hashing) in this document refers to "Number Used Once", in the context of hashing and proof of work.

If you want to read more or see a concrete example, see [What is Proof of Work? Extended Concrete Example](#what-is-proof-of-work-extended-concrete-example) at the bottom of this file.

# Overview sequence diagram

![sequence diagram](readme/sequence.png)

This diagram was created with https://app.diagrams.net/.
To edit it, download the <a download href="readme/sequence.drawio">diagram file</a> and edit it with the https://app.diagrams.net/ web application, or you may run the application from [source](https://github.com/jgraph/drawio) if you wish.

# Configuring

💥PoW! Captcha gets all of its configuration from environment variables.

#### `POW_CAPTCHA_ADMIN_API_TOKEN`

⚠️ **REQUIRED** 

This token allows control of the Admin API & allows the bearer to create, list, and revoke application tokens.

----

#### `POW_CAPTCHA_BATCH_SIZE`

💬 *OPTIONAL* default value is 1000

How many captcha challenges to return at once.

----

#### `POW_CAPTCHA_DEPRECATE_AFTER_BATCHES`

💬 *OPTIONAL* default value is 10

How many "batches-old" captcha challenges can be before being dropped from memory.

----

#### `POW_CAPTCHA_LISTEN_PORT`

💬 *OPTIONAL* default value is 2730

Which TCP port should the server listen on.

----

#### `POW_CAPTCHA_SCRYPT_CPU_AND_MEMORY_COST`

💬 *OPTIONAL* default value is 4096

Allows you to tweak how difficult each individual hash in the proof of work will be.

----

# HTTP Captcha API

#### `POST /GetChallenges?difficultyLevel=<int>`

Required Header: `Authorization: Bearer <api-token>`

Return type: `application/json` 

`GetChallenges` returns a JSON array of 1000 strings. The Captcha server will remember each one of these challeges until it is 
restarted, or until GetChallenges has been called 10 more times. Each challenge can only be used once.

The difficultyLevel parameter specifies how many bits of difficulty the challenges should have.
Each time you increase the difficultyLevel by 1, it doubles the amount of time the Proof of Work will take on average.
The recommended value is 8. A difficulty of 8 will be solved quickly by a laptop or desktop computer, and solved within 60 seconds or so by a cell phone.


#### `POST /Verify?challenge=<string>&nonce=<string>`

Required Header: `Authorization: Bearer <api-token>`

Return type: `text/plain` (error/status messages only)

`Verify` returns HTTP 200 OK only if all of the following are true:

  - This challenge was returned by `GetChallenges`.
  - `GetChallenges` hasn't been called 10 or more times since this challenge was originally returned.
  - `Verify` has not been called on this challenge before.
  - The provided hexadecimal nonce solves the challenge. 
    - (The winning nonce string will be passed to the function you specify in [data-sqr-captcha-callback](#data-sqr-captcha-callback). You just have to make sure to post it to your server so your server can include it when it calls `/Verify`)


Otherwise it returns 404, 400, or 500.


#### `GET /static/<filename>`

Return type: depends on file

Files:
  
  - captcha.js
  - captcha.css
  - proofOfWorker.js

You only need to include `captcha.js` in your page, it will pull in the other files automatically if they are not already present in the page.
See below for a more detailed implementation walkthrough.

# HTTP Admin API

#### `GET /Tokens`

Required Header: `Authorization: Bearer <admin-api-token>`

Return type: `text/plain` 

Lists all existing api tokens in CSV format, including the token itself, the name, and when it was created.

#### `POST /Tokens/Create?name=<string>`

Required Header: `Authorization: Bearer <admin-api-token>`

Return type: `text/plain`

Creates a new given API token with the given name and returns the token as a plain text hexadecimal string.

#### `POST /Tokens/Revoke?token=<api-token>`

Required Header: `Authorization: Bearer <admin-api-token>`

Return type: `text/plain`  (error/status messages only)

Revokes an existing API token.


# HTML DOM API

In order to set up 💥PoW! Captcha on your page, you just need to load/include `captcha.js` and one or more html elements 
with all 3 of the following properties:

#### `data-sqr-captcha-url`

This is the base url from which `captcha.js` will attempt to load additional resources `captcha.css` and `proofOfWorker.js`.

> 💬 *INFO* In our examples, we passed the captcha server URL down to the HTML page and used it as the value for this property.
However, that's not required. The HTML page doesn't need to talk to the captcha server at all, it just needs to know where it can
download the `captcha.css` and `proofOfWorker.js` files. There is nothing stopping you from simply hosting those files on your own server or CDN and placing the corresponding URL into the `data-sqr-captcha-url` property.

#### `data-sqr-captcha-challenge`

Set this property to one of the challenge strings returned by `GetChallenges`. It must be unique, each challenge can only be used once.

⚠️ **NOTE** that the element with the 3 `sqr-captcha-xyz` data properties **MUST** be placed **inside a form element**. This is required, to allow the  captcha to know which input elements it needs to trigger on. We only want the captcha to trigger when the user actually intends to submit the form; otherwise we are wasting a lot of their CPU cycles for no reason!

#### `data-sqr-captcha-callback`

This is the name of a function in the global namespace which will be called & passed the winning nonce once the Proof of Work 
is completed. So, for example, if you had:

`<div ... data-sqr-captcha-callback="myCallbackFunction"></div>`

Then you would provide your callback like so:

```
<script>
  window.myCallbackFunction = function(nonce) {
    ...
  }
</script>
```

> 💬 *INFO* You may also nest the callback inside object(s) if you wish: 

`<div ... data-sqr-captcha-callback="myApp.myCallbackFunction"></div>`

```
<script>
  window.myApp = {
    myCallbackFunction: function(nonce) {
      ...
    }
  };
</script>
```

When `captcha.js` runs, if it finds an element with `data-sqr-captcha-challenge` & `data-sqr-captcha-callback`, but the callback function is not defined yet, it will print a warning message. If the callback is still not defined when the Proof of Work is completed, it will throw an error. 

> 💬 *INFO* the element with the `sqr-captcha` data properties should probably be styled to have a very small font size. When I was designing the css for the captcha element, I made everything scale based on the font size (by using `em`). But because the page I was testing it on had a small font by default, I accidentally made it huge when it is rendered on a default HTML page. So for now you will want to make the font size of the element which contains it fairly small, like `10px` or `11px`. 

#### `window.sqrCaptchaInit`

After everything is set up in the DOM, this function must be called to initialize the captcha(s). This function will throw an error if it is called more than once without calling `window.sqrCaptchaReset()` in between.

For example:

```
<script>
  window.sqrCaptchaInit();
</script>
```

#### `window.sqrCaptchaReset`

Resets the captchas, stops the webworkers, etc. Use this if you have updated the page and you need to call `window.sqrCaptchaInit` again.

----

# Running the example app

The `example` folder in this repository contains an example app that demonstrates how to implement the 💥PoW! Captcha 
in as simple of a fashion as possible.

If you wish to run the example app, you will have to run both the 💥PoW! Captcha server and the example app server.

The easiest way to do this would probably be to open two separate terminal windows or tabs and run each app in its own terminal.

#### `terminal 1`
```
forest@thingpad:~/Desktop/git/sequentialread-pow-captcha$ go run main.go

panic: can't start the app, the POW_CAPTCHA_ADMIN_API_TOKEN environment variable is required

goroutine 1 [running]:
main.main()
        /home/forest/Desktop/git/sequentialread-pow-captcha/main.go:84 +0xf45
exit status 2
```
As you can see, the server requires an admin API token to be set. This is the token we will use authenticate and create
individual tokens for different apps or different people who all might want to use the captcha server.

Once we provide this admin API token environment variable, it will run just fine:

```
forest@thingpad:~/Desktop/git/sequentialread-pow-captcha$ POW_CAPTCHA_ADMIN_API_TOKEN="example_admin" go run main.go
2021/02/25 16:24:00 💥  PoW! Captcha server listening on port 2370
```

Now let's try to launch the example Todo List application:

#### `terminal 2`
```
forest@thingpad:~/Desktop/git/sequentialread-pow-captcha$ cd example/
forest@thingpad:~/Desktop/git/sequentialread-pow-captcha/example$ go run main.go

panic: can't start the app, the CAPTCHA_API_TOKEN environment variable is required

goroutine 1 [running]:
main.main()
        /home/forest/Desktop/git/sequentialread-pow-captcha/example/main.go:40 +0x488
exit status 2
```

It's a similar story for the example app, except this time we can't just make up any old token, we have to ask the captcha server to generate a new API token for the example app. I will do this by manually sending it an http request with `curl`:

```
$ curl -X POST -H "Authorization: Bearer example_admin" http://localhost:2370/Tokens/Create
400 Bad Request: url param ?name=<string> is required

$ curl -X POST -H "Authorization: Bearer example_admin" http://localhost:2370/Tokens/Create?name=todo-list
b804f221e8a9053b2e6e89de83c5d7a4
```

Now we can use this token to start the example Todo List app:

```
$ CAPTCHA_API_TOKEN="b804f221e8a9053b2e6e89de83c5d7a4" go run main.go
2021/02/25 16:38:32 📋  Todo List example application listening on port 8080
```

Then, you should be able to visit the example Todo List application in the browser at http://localhost:8080.

# Implementation walkthrough via example app

Lets walk through how example app works and how it integrates the 💥PoW! Captcha.

The Todo List app has three pieces of configuration related to the captcha: the API token, the url, and the difficulty. 
Currently the url and difficulty are hardcoded into the Todo List app's code, while the API token is provideded via an environment variable.

```
// 5 bits of difficulty, 1 in 2^5 (1 in 32) tries will succeed on average.
//
// 7 bits of difficulty would be fine for apps that are never used on mobile phones, 5 is better suited for mobile apps
//
const captchaDifficultyLevel = 5

...

  apiToken := os.ExpandEnv("$CAPTCHA_API_TOKEN")
  if apiToken == "" {
    panic(errors.New("can't start the app, the CAPTCHA_API_TOKEN environment variable is required"))
  }

  captchaAPIURL, err = url.Parse("http://localhost:2370")
```

When the Todo List app starts, it has a few procedures it runs through to ensure it's ready to run, including 
retrieving a batch of captcha challenges from the captcha API:

```
func main() {

  ...

  err = loadCaptchaChallenges()
  if err != nil {
    panic(errors.Wrap(err, "can't start the app because could not loadCaptchaChallenges():"))
  }
```

`loadCaptchaChallenges()` calls the `GetChallenges` API & sets the global variable `captchaChallenges`.

It's a good idea to do this when your app starts, to ensure that it can talk to the captcha server before it starts serving content to users.

The Todo List app only has one route: `/`.

This route displays a basic HTML page with a form, based on the template `index.html`.

```
  http.HandleFunc("/", func(responseWriter http.ResponseWriter, request *http.Request) {

    ...

  })
```

This route does 4 things:

  1. If it was a `POST` request, call the `Verify` endpoint to ensure that a valid captcha challenge and nonce were posted.
    - see `validateCaptcha` on line 202.
  2. If it was a *valid* `POST` request, add the posted `item` string to the global list variable `items`.
  3. Check if the global `captchaChallenges` list is running out, if it is, kick off a background process to grab more from the `GetChallenges` API.
    - see `loadCaptchaChallenges` on line 155.
  4. Consume one challenge string from the global `captchaChallenges` list variable and output an HTML page containing that challenge.

The captcha API (`GetChallenges` and `Verify`) was designed this way to optimize the performance of your application; instead of calling something like *GetCaptchaChallenge* for every single request, your application can load batches of captcha challenges asychronously in the background, and always have a challenge loaded into local memory & ready to go.

However, you have to make sure that you are using it right: 

 - You must ensure that you only serve each challenge once, and
 - You must only call `GetChallenges` when necessary (when you are running out of challenges). If you call it too often you may accidentally expire otherwise-valid challenges before they can be verified. 
 - Note that for high-traffic web sites where multiple requests can hit the server at once, you should probably use a [lock, mutex](https://git.sequentialread.com/forest/sequentialread-comments/src/af2f999134214412c1c6cf32c458e9b8a8c88289/main.go#L278), partitioning scheme, or other thread safe data structure to ensure that two concurrent requests don't end up trying to grab the same challenge from the list ([Software Race Condition](https://en.wikipedia.org/wiki/Race_condition#Software)).

---

Anyways, lets get on with things & look at how the Todo List app renders its HTML page. 
There are two main important parts, the form and the javascript at the bottom:

```
        <form method="POST" action="/">
          <input type="text" name="item" />
          <input type="hidden" name="challenge" value="{{ .Challenge }}" />
          <input type="hidden" name="nonce" />
          <input type="submit" disabled="true" value="Add" />
          <div class="captcha-container" 
              data-sqr-captcha-url="{{ .CaptchaURL }}" 
              data-sqr-captcha-challenge="{{ .Challenge }}" 
              data-sqr-captcha-callback="myCaptchaCallback">
         </div>
        </form>

        ...

  <script>
    window.myCaptchaCallback = (nonce) => {
      document.querySelector("form input[name='nonce']").value = nonce;
      document.querySelector("form input[type='submit']").disabled = false;
    };
  </script>
  <script src="{{ .CaptchaURL }}/static/captcha.js"></script>
```

⚠️ **NOTE** that the element with the `sqr-captcha` data properties is placed **inside a form element**. This is required because the captcha needs to know which input elements it should trigger on. We only want the captcha to trigger when the user actually intends to submit the form; otherwise we are wasting a lot of their CPU cycles for no reason!

> 💬 *INFO* The double curly brace elements like `{{ .Challenge }}` are Golang string template interpolations.  They are specific to the example app & how it renders the page.

When the page loads, the `captcha.js` script will execute, querying the page for all elements with the `data-sqr-captcha-challenge`
property. It will then validate each element to make sure it also has the `data-sqr-captcha-url` and `data-sqr-captcha-callback` properties. For each element it found, it will locate the `<form>` parent/grandparent enclosing the element. If none are found, it will throw an error. Otherwise, it will set up an event listener on every input element inside that form, so that as soon as the user starts filling out the form, the captcha display will pop up and the Proof of Work will begin. 

When the Proof of Work finishes, `captcha.js` will call the function specified by `data-sqr-captcha-callback`, passing the winning nonce as the first argument, or throw an error if that function is not defined.

> 💬 *INFO* the element with the `sqr-captcha` data properties also has a class that *WE* defined, called `captcha-container`.
This class has a very small font size. When I was designing the css for the captcha element, I made everything scale based on the font size (by using `em`). But because the page I was testing it on had a small font by default, I accidentally made it huge when it is rendered on a default HTML page. So for now you will want to make the font size of the element which contains it fairly small. 

```
<style>
    .captcha-container {
      margin-top: 1em;
      font-size: 10px;
    }
    
    ...

</style>
```

I think that concludes the walkthrough! In the Todo App, as soon as `captcha.js` calls `myCaptchaCallback`, the form will be completely filled out and the submit button will be enabled. When the form is posted, the browser will make a `POST` request to the server, and the server logic we already discussed will take over, closing the loop. 

# Implementation Details for Developers

💥PoW! Captcha uses [WebWorker](https://developer.mozilla.org/en-US/docs/Web/API/Web_Workers_API/Using_web_workers)s and [WebAssembly (WASM)](https://developer.mozilla.org/en-US/docs/WebAssembly) to calculate Proof of Work in the browser as efficiently as possible. WebWorkers allow the application to run code on multiple threads and take advantage of multi-core CPUs. WebAssembly gives us access to *actual integers* (😲) and more low-level memory operations that have been historically missing from JavaScript. 

I measured the performance of the application with and without WebWorker / WebAssembly on a variety of devices.

I tried two different implementations of the scrypt hash function, one from the [Stanford Javascript Crypto Library (sjcl)](https://github.com/bitwiseshiftleft/sjcl) and the WASM one from [github.com/MyEtherWallet/scrypt-wasm](https://github.com/MyEtherWallet/scrypt-wasm).

| hardware | sjcl,single thread | sjcl,multi-thread | WASM,multi-thread |
| :------------- | :------------- | :----------: | -----------: |
| Lenovo T480s | 1-2 h/s | ~5 h/s  | ~70 h/s  |
| Motorolla G7  | not tested | not tested | ~12 h/s |
| Macbook Air 2018  | not tested | not tested | ~ 32h/s |
| Google Pixel 3a | not tested | not tested | ~ 24h/s |

I had some trouble getting the WASM module loaded properly inside the WebWorkers. In my production environment, the web application server and the captcha server are running on separate subdomains, so I was getting cross-origin security violation issues. 

I ended up embedding the WASM binary inside the WebWorker javascript `proofOfWorker.js` using a boutique binary encoding called [base32768](https://github.com/qntm/base32768). I set up a custom build process for this in the `wasm_build` folder. It even includes the scripts necessary to clone the github.com/MyEtherWallet/scrypt-wasm repo and install the Rust compiler! You are welcome! However, this script does assume that you are running on a Linux computer. I have not tested it outside of Linux.


# What is Proof of Work? Extended Concrete Example


When you calculate the hash of a file or a piece of data, you get this random string of characters:

```
forest@thingpad:~/Desktop/git/sequentialread-pow-captcha$ sha256sum LICENSE.md 
119ba12858fcf041fc43bb3331eaeaf313e1d01e278d5cc911fd2c60dc1c503f  LICENSE.md
```

Here, I have called the SHA256 hash function on the GPLv3 `LICENSE.md` file in this repo. The result is displayed as a hexidecimal string, that is, each character can have one of 16 possible values, 0-9 and a-f. You can think of it like rolling a whole bunch of 16-sided dice, however, it's not random like dice are, its *pseudorandom*, meaning that given the same input file, if we execute the same hash function multiple times, it will return the same output. All the dice will land the same way every time:

```
forest@thingpad:~/Desktop/git/sequentialread-pow-captcha$ sha256sum LICENSE.md 
119ba12858fcf041fc43bb3331eaeaf313e1d01e278d5cc911fd2c60dc1c503f  LICENSE.md

forest@thingpad:~/Desktop/git/sequentialread-pow-captcha$ sha256sum LICENSE.md 
119ba12858fcf041fc43bb3331eaeaf313e1d01e278d5cc911fd2c60dc1c503f  LICENSE.md

forest@thingpad:~/Desktop/git/sequentialread-pow-captcha$ sha256sum LICENSE.md 
119ba12858fcf041fc43bb3331eaeaf313e1d01e278d5cc911fd2c60dc1c503f  LICENSE.md
```

However, If I change the input, even if I only change it a tiny bit, say, append the letter `a` at the end of the file, it will completely change the way the result shakes out:

```
# append the letter a to the end of the file
forest@thingpad:~/Desktop/git/sequentialread-pow-captcha$ echo 'a' >> LICENSE.md 

# calculate the SHA256 hash again
forest@thingpad:~/Desktop/git/sequentialread-pow-captcha$ sha256sum LICENSE.md 
67e0e2cc3429b799036bfa95e2bd7854a0e468939d6cb9d4a3e9d32c3b6615dc  LICENSE.md
```

It's impossible to tell how the hash will be affected by changing the input... Well, unless you calculate the hash! 
This is related to the famous [Halting Problem](https://en.wikipedia.org/wiki/Halting_problem) from computer science. 

PoW is a game which exploits these interesting properties of hash functions. It works like this: I give you a file, and then you have to change the file (Add "`a`"s at the end, increment a number in the file, whatever you want to do) and recalculate the hash each time you change it, until you find a hash which ends in two zeros in a row. Or three zeros in a row, or four, whatever. Since there are 16 possible values for each character, each additional required zero divides your likelhood of finding the "winning" hash by 16.

The number or string of "`a`"s, whatever it is you use to change the file before you hash it, is called the Nonce.

This is exactly how Bitcoin mining works, Bitcoin requires miners to search for SHA256 hashes that end in a rediculously unlikely number of zeros, like flipping 100 coins and getting 100 heads in a row.

💥PoW! Captcha uses a different hash function called [Scrypt](https://en.wikipedia.org/wiki/Scrypt). Scrypt was designed to take an arbitrarily long amount of time to execute on a computer, and to be hard to optimize.

A modified version of Scrypt is used by the crypto currency [Litecoin](https://en.wikipedia.org/wiki/Litecoin).

Like I mentioned in the condensed "What is Proof of Work" section, because of this pseudorandom behaviour, we can't predict how long a given captcha will take to complete. The UI does have a "progress bar" but the behaviour of the bar is more related to probability than to progress. In fact, it displays the "probability that we should have found the answer already", which is related to the amount of work done so far, but it's not exactly a linear relationship. 

Here is a screenshot of a plot I generated using WolframAlpha while I was developing this progress bar, given the formula for the progress bar's width:

![wolfram alpha plot](readme/probability.png)

This explains why the progress bar moves faster at the start & slows down once it starts approaching the end.
