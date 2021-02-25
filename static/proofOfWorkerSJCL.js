// output of: 
// git clone https://github.com/bitwiseshiftleft/sjcl
// cd sjcl
// ./configure --without-all --with-scrypt --with-codecBase64 --with-codecHex
// make
// cat sjcl.js

"use strict";var sjcl={cipher:{},hash:{},keyexchange:{},mode:{},misc:{},codec:{},exception:{corrupt:function(b){this.toString=function(){return"CORRUPT: "+this.message};this.message=b},invalid:function(b){this.toString=function(){return"INVALID: "+this.message};this.message=b},bug:function(b){this.toString=function(){return"BUG: "+this.message};this.message=b},notReady:function(b){this.toString=function(){return"NOT READY: "+this.message};this.message=b}}};
sjcl.bitArray={bitSlice:function(b,d,c){b=sjcl.bitArray.m(b.slice(d/32),32-(d&31)).slice(1);return void 0===c?b:sjcl.bitArray.clamp(b,c-d)},extract:function(b,d,c){var a=Math.floor(-d-c&31);return((d+c-1^d)&-32?b[d/32|0]<<32-a^b[d/32+1|0]>>>a:b[d/32|0]>>>a)&(1<<c)-1},concat:function(b,d){if(0===b.length||0===d.length)return b.concat(d);var c=b[b.length-1],a=sjcl.bitArray.getPartial(c);return 32===a?b.concat(d):sjcl.bitArray.m(d,a,c|0,b.slice(0,b.length-1))},bitLength:function(b){var d=b.length;return 0===
d?0:32*(d-1)+sjcl.bitArray.getPartial(b[d-1])},clamp:function(b,d){if(32*b.length<d)return b;b=b.slice(0,Math.ceil(d/32));var c=b.length;d=d&31;0<c&&d&&(b[c-1]=sjcl.bitArray.partial(d,b[c-1]&2147483648>>d-1,1));return b},partial:function(b,d,c){return 32===b?d:(c?d|0:d<<32-b)+0x10000000000*b},getPartial:function(b){return Math.round(b/0x10000000000)||32},equal:function(b,d){if(sjcl.bitArray.bitLength(b)!==sjcl.bitArray.bitLength(d))return!1;var c=0,a;for(a=0;a<b.length;a++)c|=b[a]^d[a];return 0===
c},m:function(b,d,c,a){var e;e=0;for(void 0===a&&(a=[]);32<=d;d-=32)a.push(c),c=0;if(0===d)return a.concat(b);for(e=0;e<b.length;e++)a.push(c|b[e]>>>d),c=b[e]<<32-d;e=b.length?b[b.length-1]:0;b=sjcl.bitArray.getPartial(e);a.push(sjcl.bitArray.partial(d+b&31,32<d+b?c:a.pop(),1));return a},s:function(b,d){return[b[0]^d[0],b[1]^d[1],b[2]^d[2],b[3]^d[3]]},byteswapM:function(b){var d,c;for(d=0;d<b.length;++d)c=b[d],b[d]=c>>>24|c>>>8&0xff00|(c&0xff00)<<8|c<<24;return b}};
sjcl.codec.utf8String={fromBits:function(b){var d="",c=sjcl.bitArray.bitLength(b),a,e;for(a=0;a<c/8;a++)0===(a&3)&&(e=b[a/4]),d+=String.fromCharCode(e>>>8>>>8>>>8),e<<=8;return decodeURIComponent(escape(d))},toBits:function(b){b=unescape(encodeURIComponent(b));var d=[],c,a=0;for(c=0;c<b.length;c++)a=a<<8|b.charCodeAt(c),3===(c&3)&&(d.push(a),a=0);c&3&&d.push(sjcl.bitArray.partial(8*(c&3),a));return d}};
sjcl.codec.hex={fromBits:function(b){var d="",c;for(c=0;c<b.length;c++)d+=((b[c]|0)+0xf00000000000).toString(16).substr(4);return d.substr(0,sjcl.bitArray.bitLength(b)/4)},toBits:function(b){var d,c=[],a;b=b.replace(/\s|0x/g,"");a=b.length;b=b+"00000000";for(d=0;d<b.length;d+=8)c.push(parseInt(b.substr(d,8),16)^0);return sjcl.bitArray.clamp(c,4*a)}};
sjcl.codec.base64={i:"ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789+/",fromBits:function(b,d,c){var a="",e=0,f=sjcl.codec.base64.i,k=0,g=sjcl.bitArray.bitLength(b);c&&(f=f.substr(0,62)+"-_");for(c=0;6*a.length<g;)a+=f.charAt((k^b[c]>>>e)>>>26),6>e?(k=b[c]<<6-e,e+=26,c++):(k<<=6,e-=6);for(;a.length&3&&!d;)a+="=";return a},toBits:function(b,d){b=b.replace(/\s|=/g,"");var c=[],a,e=0,f=sjcl.codec.base64.i,k=0,g;d&&(f=f.substr(0,62)+"-_");for(a=0;a<b.length;a++){g=f.indexOf(b.charAt(a));
if(0>g)throw new sjcl.exception.invalid("this isn't base64!");26<e?(e-=26,c.push(k^g>>>e),k=g<<32-e):(e+=6,k^=g<<32-e)}e&56&&c.push(sjcl.bitArray.partial(e&56,k,1));return c}};sjcl.codec.base64url={fromBits:function(b){return sjcl.codec.base64.fromBits(b,1,1)},toBits:function(b){return sjcl.codec.base64.toBits(b,1)}};
sjcl.codec.bytes={fromBits:function(b){var d=[],c=sjcl.bitArray.bitLength(b),a,e;for(a=0;a<c/8;a++)0===(a&3)&&(e=b[a/4]),d.push(e>>>24),e<<=8;return d},toBits:function(b){var d=[],c,a=0;for(c=0;c<b.length;c++)a=a<<8|b[c],3===(c&3)&&(d.push(a),a=0);c&3&&d.push(sjcl.bitArray.partial(8*(c&3),a));return d}};sjcl.hash.sha256=function(b){this.g[0]||r(this);b?(this.f=b.f.slice(0),this.c=b.c.slice(0),this.a=b.a):this.reset()};sjcl.hash.sha256.hash=function(b){return(new sjcl.hash.sha256).update(b).finalize()};
sjcl.hash.sha256.prototype={blockSize:512,reset:function(){this.f=this.l.slice(0);this.c=[];this.a=0;return this},update:function(b){"string"===typeof b&&(b=sjcl.codec.utf8String.toBits(b));var d,c=this.c=sjcl.bitArray.concat(this.c,b);d=this.a;b=this.a=d+sjcl.bitArray.bitLength(b);if(0x1fffffffffffff<b)throw new sjcl.exception.invalid("Cannot hash more than 2^53 - 1 bits");if("undefined"!==typeof Uint32Array){var a=new Uint32Array(c),e=0;for(d=512+d-(512+d&0x1ff);d<=b;d+=512)u(this,a.subarray(16*e,
16*(e+1))),e+=1;c.splice(0,16*e)}else for(d=512+d-(512+d&0x1ff);d<=b;d+=512)u(this,c.splice(0,16));return this},finalize:function(){var b,d=this.c,c=this.f,d=sjcl.bitArray.concat(d,[sjcl.bitArray.partial(1,1)]);for(b=d.length+2;b&15;b++)d.push(0);d.push(Math.floor(this.a/0x100000000));for(d.push(this.a|0);d.length;)u(this,d.splice(0,16));this.reset();return c},l:[],g:[]};
function u(b,d){var c,a,e,f=b.f,k=b.g,g=f[0],h=f[1],l=f[2],n=f[3],m=f[4],q=f[5],p=f[6],t=f[7];for(c=0;64>c;c++)16>c?a=d[c]:(a=d[c+1&15],e=d[c+14&15],a=d[c&15]=(a>>>7^a>>>18^a>>>3^a<<25^a<<14)+(e>>>17^e>>>19^e>>>10^e<<15^e<<13)+d[c&15]+d[c+9&15]|0),a=a+t+(m>>>6^m>>>11^m>>>25^m<<26^m<<21^m<<7)+(p^m&(q^p))+k[c],t=p,p=q,q=m,m=n+a|0,n=l,l=h,h=g,g=a+(h&l^n&(h^l))+(h>>>2^h>>>13^h>>>22^h<<30^h<<19^h<<10)|0;f[0]=f[0]+g|0;f[1]=f[1]+h|0;f[2]=f[2]+l|0;f[3]=f[3]+n|0;f[4]=f[4]+m|0;f[5]=f[5]+q|0;f[6]=f[6]+p|0;f[7]=
f[7]+t|0}function r(b){function d(a){return 0x100000000*(a-Math.floor(a))|0}for(var c=0,a=2,e,f;64>c;a++){f=!0;for(e=2;e*e<=a;e++)if(0===a%e){f=!1;break}f&&(8>c&&(b.l[c]=d(Math.pow(a,.5))),b.g[c]=d(Math.pow(a,1/3)),c++)}}sjcl.misc.hmac=function(b,d){this.j=d=d||sjcl.hash.sha256;var c=[[],[]],a,e=d.prototype.blockSize/32;this.b=[new d,new d];b.length>e&&(b=d.hash(b));for(a=0;a<e;a++)c[0][a]=b[a]^909522486,c[1][a]=b[a]^1549556828;this.b[0].update(c[0]);this.b[1].update(c[1]);this.h=new d(this.b[0])};
sjcl.misc.hmac.prototype.encrypt=sjcl.misc.hmac.prototype.mac=function(b){if(this.o)throw new sjcl.exception.invalid("encrypt on already updated hmac called!");this.update(b);return this.digest(b)};sjcl.misc.hmac.prototype.reset=function(){this.h=new this.j(this.b[0]);this.o=!1};sjcl.misc.hmac.prototype.update=function(b){this.o=!0;this.h.update(b)};sjcl.misc.hmac.prototype.digest=function(){var b=this.h.finalize(),b=(new this.j(this.b[1])).update(b).finalize();this.reset();return b};
sjcl.misc.pbkdf2=function(b,d,c,a,e){c=c||1E4;if(0>a||0>c)throw new sjcl.exception.invalid("invalid params to pbkdf2");"string"===typeof b&&(b=sjcl.codec.utf8String.toBits(b));"string"===typeof d&&(d=sjcl.codec.utf8String.toBits(d));e=e||sjcl.misc.hmac;b=new e(b);var f,k,g,h,l=[],n=sjcl.bitArray;for(h=1;32*l.length<(a||1);h++){e=f=b.encrypt(n.concat(d,[h]));for(k=1;k<c;k++)for(f=b.encrypt(f),g=0;g<f.length;g++)e[g]^=f[g];l=l.concat(e)}a&&(l=n.clamp(l,a));return l};
sjcl.misc.scrypt=function(b,d,c,a,e,f,k){var g=Math.pow(2,32)-1,h=sjcl.misc.scrypt;c=c||16384;a=a||8;e=e||1;if(a*e>=Math.pow(2,30))throw sjcl.exception.invalid("The parameters r, p must satisfy r * p < 2^30");if(2>c||c&0!=c-1)throw sjcl.exception.invalid("The parameter N must be a power of 2.");if(c>g/128/a)throw sjcl.exception.invalid("N too big.");if(a>g/128/e)throw sjcl.exception.invalid("r too big.");d=sjcl.misc.pbkdf2(b,d,1,128*e*a*8,k);a=d.length/e;h.reverse(d);for(g=0;g<e;g++){var l=d.slice(g*
a,(g+1)*a);h.blockcopy(h.ROMix(l,c),0,d,g*a)}h.reverse(d);return sjcl.misc.pbkdf2(b,d,1,f,k)};
sjcl.misc.scrypt.salsa20Core=function(b,d){function c(a,b){return a<<b|a>>>32-b}for(var a=b.slice(0),e=d;0<e;e-=2)a[4]^=c(a[0]+a[12],7),a[8]^=c(a[4]+a[0],9),a[12]^=c(a[8]+a[4],13),a[0]^=c(a[12]+a[8],18),a[9]^=c(a[5]+a[1],7),a[13]^=c(a[9]+a[5],9),a[1]^=c(a[13]+a[9],13),a[5]^=c(a[1]+a[13],18),a[14]^=c(a[10]+a[6],7),a[2]^=c(a[14]+a[10],9),a[6]^=c(a[2]+a[14],13),a[10]^=c(a[6]+a[2],18),a[3]^=c(a[15]+a[11],7),a[7]^=c(a[3]+a[15],9),a[11]^=c(a[7]+a[3],13),a[15]^=c(a[11]+a[7],18),a[1]^=c(a[0]+a[3],7),a[2]^=
c(a[1]+a[0],9),a[3]^=c(a[2]+a[1],13),a[0]^=c(a[3]+a[2],18),a[6]^=c(a[5]+a[4],7),a[7]^=c(a[6]+a[5],9),a[4]^=c(a[7]+a[6],13),a[5]^=c(a[4]+a[7],18),a[11]^=c(a[10]+a[9],7),a[8]^=c(a[11]+a[10],9),a[9]^=c(a[8]+a[11],13),a[10]^=c(a[9]+a[8],18),a[12]^=c(a[15]+a[14],7),a[13]^=c(a[12]+a[15],9),a[14]^=c(a[13]+a[12],13),a[15]^=c(a[14]+a[13],18);for(e=0;16>e;e++)b[e]=a[e]+b[e]};
sjcl.misc.scrypt.blockMix=function(b){for(var d=b.slice(-16),c=[],a=b.length/16,e=sjcl.misc.scrypt,f=0;f<a;f++)e.blockxor(b,16*f,d,0,16),e.salsa20Core(d,8),0==(f&1)?e.blockcopy(d,0,c,8*f):e.blockcopy(d,0,c,8*(f^1+a));return c};sjcl.misc.scrypt.ROMix=function(b,d){for(var c=b.slice(0),a=[],e=sjcl.misc.scrypt,f=0;f<d;f++)a.push(c.slice(0)),c=e.blockMix(c);for(f=0;f<d;f++)e.blockxor(a[c[c.length-16]&d-1],0,c,0),c=e.blockMix(c);return c};
sjcl.misc.scrypt.reverse=function(b){for(var d in b){var c=b[d]&255,c=c<<8|b[d]>>>8&255,c=c<<8|b[d]>>>16&255,c=c<<8|b[d]>>>24&255;b[d]=c}};sjcl.misc.scrypt.blockcopy=function(b,d,c,a,e){var f;e=e||b.length-d;for(f=0;f<e;f++)c[a+f]=b[d+f]|0};sjcl.misc.scrypt.blockxor=function(b,d,c,a,e){var f;e=e||b.length-d;for(f=0;f<e;f++)c[a+f]=c[a+f]^b[d+f]|0};"undefined"!==typeof module&&module.exports&&(module.exports=sjcl);"function"===typeof define&&define([],function(){return sjcl});

let working = false;
const batchSize = 2;

onmessage = function(e) {
  if(e.data.stop) {
    working = false;
    return;
  }

  const challengeBase64 = e.data.challenge;
  const workerId = e.data.workerId;
  if(!challengeBase64) {
    postMessage({
      type: "error", 
      challenge: challengeBase64,
      message: `challenge was not provided`
    });
  }
  working = true;
  let challengeJSON = null;
  let challenge = null;
  try {
    challengeJSON = atob(challengeBase64);
  } catch (err) {
    postMessage({
      type: "error", 
      challenge: challengeBase64,
      message: `couldn't decode challenge '${challengeBase64}' as base64: ${err}`
    });
  }
  try {
    challenge = JSON.parse(challengeJSON);
  } catch (err) {
    postMessage({
      type: "error", 
      challenge: challengeBase64,
      message: `couldn't parse challenge '${challengeJSON}' as json: ${err}`
    });
  }

  challenge = {
    cpuAndMemoryCost: challenge.N,
    blockSize:        challenge.r,
    paralellization:  challenge.p,
    keyLength:        challenge.klen,
    preimage:         challenge.i,
    difficulty:       challenge.d,
    difficultyLevel:  challenge.dl
  }
  
  const preimageBits = sjcl.codec.base64.toBits(challenge.preimage);
  const probabilityOfFailurePerAttempt = 1-(1/Math.pow(2, challenge.difficultyLevel));

  var i = workerId * Math.pow(2, challenge.difficulty.length) * 100;
  let smallestHash = challenge.difficulty.split("").map(x => "f").join("");

  postMessage({
    type: "progress", 
    challenge: challengeBase64,
    attempts: 0,
    smallestHash: smallestHash,
    difficulty: challenge.difficulty,
    probabilityOfFailurePerAttempt: probabilityOfFailurePerAttempt
  });

  const doWork = () => {

    var j = 0;
    while(j < batchSize) {
      j++;
      i++;
      
      const nonceBits = sjcl.codec.hex.toBits(i.toString(16))
  
      const hashBits = sjcl.misc.scrypt(
        nonceBits, 
        preimageBits, 
        challenge.cpuAndMemoryCost, 
        challenge.blockSize, 
        challenge.paralellization, 
        challenge.keyLength*8
      );
      const hashHex = sjcl.codec.hex.fromBits(hashBits);
      // if(j == 10) {
      //   console.log(workerId, hashHex);
      // }
  
      const endOfHash = hashHex.substring(hashHex.length-challenge.difficulty.length);
      if(endOfHash < smallestHash) {
        smallestHash = endOfHash
      }
      if(endOfHash <= challenge.difficulty) {
        postMessage({
          type: "success", 
          challenge: challengeBase64,
          nonce: i.toString(16),
          smallestHash: endOfHash,
          difficulty: challenge.difficulty
        });
        break
      }
    }

    if(working) {
      this.setTimeout(doWork, 1);
    }

    postMessage({
      type: "progress", 
      challenge: challengeBase64,
      attempts: batchSize,
      smallestHash: smallestHash,
      difficulty: challenge.difficulty,
      probabilityOfFailurePerAttempt: probabilityOfFailurePerAttempt
    });
  };

  doWork();  
}