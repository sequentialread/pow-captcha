(function(undefined, document){

  const challenges = Array.from(document.querySelectorAll("[data-sqr-captcha-challenge]"));
  const challengesMap = {};
  const url = null;
  let proofOfWorker = { postMessage: () => console.error("error: proofOfWorker was never loaded. ") };
  challenges.forEach(element => {

    if(!url) {
      if(!element.dataset.sqrCaptchaUrl) {
        console.error("error: element with data-sqr-captcha-challenge property is missing the data-sqr-captcha-url property");
      }
      url = element.dataset.sqrCaptchaUrl;
      if(!url.endsWith("/")) {
        url = `${url}/`
      }
    }

    let form = null;
    let parent = element.parentElement;
    let sanity = 1000;
    while(parent && !form && sanity > 0) {
      sanity--;
      if(parent.tagName == "form") {
        form = parent
      }
      parent = parent.parentElement
    }
    if(!form) {
      console.error("error: element with data-sqr-captcha-challenge property was not inside a form element");
      //todo
    }

    const onFormWasTouched = () => {
      if(!challengesMap[element.dataset.sqrCaptchaChallenge]) {
        challengesMap[element.dataset.sqrCaptchaChallenge] = element;
        myWorker.postMessage(element.dataset.sqrCaptchaChallenge);
      }
    };

    const inputElements = Array.from(form.querySelectorAll("input"))
                    .concat(Array.from(form.querySelectorAll("textarea")));
    
    inputElements.forEach(inputElement => {
      inputElement.onchange = onFormWasTouched;
      inputElement.onkeydown = onFormWasTouched;
    });
  });

  if (!window.Worker) {
    console.error("error: webworker is not support");
    //todo
  }

  if(url) {
    proofOfWorker = new Worker(`${url}static/proofOfWorker.js`);

    proofOfWorker.onmessage = function(e) {
      const challengeElement = challengesMap[e.data.challenge]
      if(!challengeElement) {
        console.error(`error: webworker sent message with unknown challenge '${e.data.challenge}'`);
      }
      if(e.data.type == "progress") {
         console.log("progress: " + e.data.value)
      } else if(e.data.type == "success") {
        console.log("success: " + e.data.nonce)
      } else if(e.data.type == "error") {
        console.error(`error: webworker errored out: '${e.data.message}'`);
      } else {
        console.error(`error: webworker sent message with unknown type '${e.data.type}'`);
      }
    };
  }


})(document);