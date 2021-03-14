import urllib3
import json
import hashlib
import re

from skimage.io import imread, imsave
from skimage.metrics import structural_similarity as ssim
from skimage.transform import resize

import base64
import operator
import os

from Crypto.Signature import pkcs1_15
from Crypto.Hash import SHA256
from Crypto.PublicKey import RSA
from Crypto.Random import get_random_bytes
from Crypto.Cipher import AES, PKCS1_OAEP, PKCS1_v1_5
import time


class CookieJar:
    def __init__(self):
        self._cookies = {}

    def __str__(self):
        s = "; ".join([f"{k}={v}" for k, v in self._cookies.items()])
        print(f"Cookies: {s}")
        return s

    def update(self, response: urllib3.response):
        c = response.headers.get("Set-Cookie", "")
        for pair in re.findall("[^\s;,]+=[^\s;,]+", c):
            k, v = pair.split("=")
            if k.lower() in ["path", "domain"]:
                continue
            if k not in self._cookies:
                print("cookie.new ", end="")
            elif self._cookies[k] != v:
                print("cookie.update ", end="")
            else:
                continue
            print(f"{k}={v}")
            self._cookies[k] = v

    def add(self, name, value):
        self._cookies[name] = value


def pinpad_imgs(jar: CookieJar):
    http = urllib3.PoolManager()

    req = http.request(
        "GET",
        "https://www.ing.com.au/KeypadService/v1/KeypadService.svc/json/PinpadImages",
        headers={"Cookie": str(jar)},
    )
    data = json.loads(req.data)
    jar.update(req)
    return data["KeypadImages"], data["PemEncryptionKey"], data["Secret"]


def encyrpt(key, data):
    data = data.encode("utf-8")
    recipient_key = RSA.import_key(key)
    # Encrypt the session key with the public RSA key
    cipher_rsa = PKCS1_v1_5.new(recipient_key)
    enc_session_key = cipher_rsa.encrypt(data)
    return base64.standard_b64encode(enc_session_key).decode("utf-8")


def solve_imgs(imgs):
    result = {}
    for i, v in enumerate(imgs):
        n = "test.png"
        with open(n, "wb") as o:
            o.write(base64.decodebytes(v.encode("utf-8")))
        s = {}
        for z in range(10):
            s[str(z)] = compare(n, f"ing-{z}.png")
        m, x = max(s.items(), key=operator.itemgetter(1))
        result[str(m)] = i
    return result


def position(solved):
    return ",".join([str(solved[str(i)]) for i in os.environ["PIN"]])


def compare(img1, img2):
    im1 = imread(img1)
    # im1 = resize(im1, (220, 360))
    im1 = im1[37:70, 77:103]
    imsave("resize.png", im1)
    im2 = imread(img2)
    im2 = im2[37:70, 77:103]
    # im2 = resize(im2, (220, 360))
    similarity = ssim(im1, im2, multichannel=True)
    return similarity


def message_sig(rsa, data):
    # print(f'Signature:"{data}"')
    h = SHA256.new(data.encode("utf-8"))
    sig = pkcs1_15.new(rsa).sign(h)
    return sig.hex(), hex(rsa.n)[2:]


def issue(headers, jar):
    u = "https://www.ing.com.au/STSServiceB2C/V1/SecurityTokenServiceProxy.svc/issue"
    http = urllib3.PoolManager()

    r = http.request("POST", u, headers=headers, body="{}".encode("utf-8"))
    jar.update(r)
    return json.loads(r.data), r.headers


def post(token, body, url, jar):
    http = urllib3.PoolManager()
    sig, _ = message_sig(rsa, f"X-AuthToken:{token}{{}}")
    headers = {
        "x-authtoken": token,
        "x-messagesignature": sig,
        "content-type": "application/json",
        "accept": "application/json",
        "User-Agent": "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/88.0.4324.152 Safari/537.36",
        "Sec-GPC": "1",
        "Origin": "https://www.ing.com.au",
        "Sec-Fetch-Site": "same-origin",
        "Sec-Fetch-Mode": "cors",
        "Sec-Fetch-Dest": "empty",
        "Referer": "https://www.ing.com.au/securebanking/",
        "Cookie": str(jar),
        "Accept-Encoding": "gzip, deflate, br",
        "Accept-Language": "en-US,en;q=0.9,pt;q=0.8",
        "Cache-Control": "no-cache",
        "Connection": "keep-alive",
        "sec-ch-ua": '"Chromium";v="88", "Google Chrome";v="88", ";Not A Brand";v="99"',
        "sec-ch-ua-mobile": "?0",
    }
    r = http.request("POST", url, headers=headers, body=body.encode("utf-8"))
    jar.update(r)
    try:
        return json.loads(r.data)
    except:
        print(r.status, r.data.decode("utf-8"))


def get_request(url, jar):
    h = urllib3.PoolManager()
    r = h.request("GET", url, headers={"Cookie": str(jar)})
    jar.update(r)


pre_urls = [
    "https://www.ing.com.au/ing-style/img/logos/Osko.png",
    "https://www.ing.com.au/ing-style/fonts/INGMe/Regular/INGMeWeb-Regular.woff",
    "https://www.ing.com.au/static/ing-help-support/kb-mapping.json",
    "https://www.ing.com.au/ing-style/fonts/INGMe/Regular/INGMeWeb-Regular.woff",
    "https://www.ing.com.au/ing-style/fonts/icomoon/icomoon.woff?-hzjjiq",
    "https://www.ing.com.au/ing-style/fonts/INGMe/Bold/INGMeWeb-Bold.woff",
    "https://www.ing.com.au/static/cms-content/html/login/ing-login-content.html",
    "https://www.ing.com.au/static/cms-content/html/logged-out/ing-logged-out-content.html",
    "https://www.ing.com.au/static/cms-content/html/header/ing-header-content.html",
    "https://www.ing.com.au/static/cms-content/html/footer/ing-footer-i18n.html",
    "https://www.ing.com.au/static/cms-content/html/footer/ing-footer-styles.html",
    "https://www.ing.com.au/static/cms-content/html/login/ing-login-content-i18n.html",
    "https://www.ing.com.au/static/cms-content/html/logged-out/ing-logged-out-content-i18n.html",
    "https://www.ing.com.au/static/cms-content/html/logged-out/ing-logged-out-content-styles.html",
    "https://www.ing.com.au/static/cms-content/html/header/ing-header-content-i18n.html",
    "https://www.ing.com.au/static/cms-content/html/header/ing-header-content-styles.html",
    "https://www.ing.com.au/static/cms-content/js/login/ing-login-content-i18n.js",
    "https://www.ing.com.au/static/cms-content/js/login/ing-login-content.js",
    "https://www.ing.com.au/static/cms-content/js/logged-out/ing-logged-out-content-i18n.js",
    "https://www.ing.com.au/static/cms-content/js/logged-out/ing-logged-out-content.js",
    "https://www.ing.com.au/static/cms-content/js/footer/ing-footer-i18n.js",
    "https://www.ing.com.au/static/cms-content/js/footer/ing-footer-content.js",
    "https://www.ing.com.au/ing-style/img/logos/Logo-footer-public@2x.png",
    "https://www.ing.com.au/ing-style/img/logos/Logo-footer@2x.png",
    "https://www.ing.com.au/static/cms-content/js/header/ing-header-content-i18n.js",
    "https://www.ing.com.au/ing-style/fonts/ing-icon-font/ing-icon-font.woff?-hzjjiq",
    "https://www.ing.com.au/static/cms-content/js/header/ing-header-content.js",
    "https://www.ing.com.au/ing-style/img/logos/Logo-sm@2x.png",
    "https://www.ing.com.au/ing-style/img/logos/Logo-header@2x.png",
    "https://www.ing.com.au/ing-style/img/logos/lion-white.png",
]
post_urls = [
    "https://www.ing.com.au/ing-style/img/logos/Logo-footer@2x.png",
    "https://www.ing.com.au/static/cms-content/html/product-pages/insurance/ing-dashboard-banner-insurance.html",
]

rsa = RSA.generate(1024, e=10001)


def main():
    cookies = CookieJar()
    raw_imgs, pem, secret = pinpad_imgs(cookies)
    imgs = solve_imgs(raw_imgs)
    data = position(imgs)
    enc = encyrpt(pem, data)
    sig, mod = message_sig(rsa, "X-AuthToken:{}")
    headers = {
        "x-AuthCIF": "25277007",
        "x-AuthPIN": enc,
        "x-AuthSecret": secret,
        "x-AuthToken": "",
        "x-MessageSignature": sig,
        "x-MessageSignKey": mod,
        "content-type": "application/json",
        "Cookie": str(cookies),
    }
    for u in pre_urls:
        get_request(
            u,
            cookies,
        )

    login, headers = issue(headers, cookies)

    for u in post_urls:
        get_request(
            u,
            cookies,
        )

    body = "{}"

    cookies.add("puid", "9962aea0-586b-4b4d-b4b8-76ca5355eed3")
    payload = post(
        login["Token"],
        body,
        "https://www.ing.com.au/api/Dashboard/Service/DashboardService.svc/json/Dashboard/loaddashboard",
        cookies,
    )


if __name__ == "__main__":
    main()
