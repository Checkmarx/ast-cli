source = ["./dist/cx-win/cx.exe"]
bundle_id = "com.checkmarx.cli"

sign:
  private_key: |
      -----BEGIN RSA PRIVATE KEY-----
      MIIEvQIBADANBgkqhkiG9w0BAQEFAASCBKcwggSjAgEAAoIBAQCrsovoXIXsMtgE
      KGbCH+6K9xtoD4Yy6aGmHjs+32ziIcpplPXkvI/wvIU7D1bmCNlS/2H1R/F0VYKk
      bZWZ//3xqBKANS4wGqTrSC++/0HIDrE93EpENREdPFDwKavwVdqj5PdxnUWYjPhI
      7qdtM+O8ZELymjEDDhVcDNo8K5au5eWP/xYLHE+lRpLhzfzQd79G+xG+hqgn+Lnt
      Lywh+Vksy+rQ0Z+CvjwFCQbDut0MHJYXxrO6nAMChmKO3GXmvOb18BAEWnhXjCFb
      7VgC8JoNUyNtjBxia/hAKVFMjYx4LfiKzIcxzhkoJ8BMaNyjh+EL5hFCb9P+J601
      mJWuy7xxAgMBAAECggEAAPa6PBJL4qbo6UIQTJnpCQDo15lRtaaz1HbCOqC+r9jE
      dfoC9NcdoDpwrYORJ26oiKOcGUg/edmSh4mBb9k8486/ltZllVnK7/KqaPIuHHk/
      o7MhPBeHqnA4nJaBS3Kx7N5XyLybI8dzy9YCHNXwGvI9oXa93HBnbIo6beDJQl9P
      42SQduHAcAVEKJ+J+gk6rnM5qtCtmCBGRFzSlD2/GT1wNGvYju8+Hdj8camOpzki
      ZBVsw+ZFZVYQqYTiUe0xMy6oMrT2rtbDV+Plp+iRLBP4e5u7B42HPgf/6bzZOaWC
      IiUb5cQW+0Aw4YTb/5mNRgITf6ZfcpvDBrFBY+PhgQKBgQDSiS9nunbi1wlH77ym
      69KeHy7HBt42LyFPmsWEr4SQS2/ujoPnrZL9yiiEJ8CCvrXFDLg4YZFxA99vlCI0
      yHfGwZSq5bMCJzT04pl3XMDkvTQGvX9eupjehXHc2y4GlH2iDKJwbx9wzYiKwOFG
      ghcHGfAZLrOv4OGvLMUjBnuNwQKBgQDQxkz0B0DoZbbcjNo0LWVkLxUxB0VYGXcy
      NR9+3gFg8P27Hf6/uiryULncA8Bf2Sl/u8Q2wES7f9qIcyu70aB1zJheV7EFEgUb
      JN7vqLpO1FP1DdiglDZoPIFjZwOEgEZ0e7vvWxc3eFaSNo0N8Fh/FwjXIENsaazj
      4i6jh4E6sQKBgHEcdzWZfon80eWuLYLYq/1771vKmtQtmg30ry3MRsJnZSmbs85i
      +NgVJpNp8AnOgEXvwYG5GbTISeDei0okcgV8t2zhn70GZ3Mx0xXH5XJ/HFaKtMWm
      Jr9Wnofz0dSDLsRDWXpimVe3dSZm3iFNfyW3j8FXz/4sKdQ9j2Rz9SmBAoGAUEiR
      ex329ed3ZGS93GbAoMACVDJJllFkpugKzoys1wyVZglo123N6hTlBBhlN/aYoMgh
      8jQJuli2PtabMMSyAdrFlTH/nsWJNSD+ogaubnX0Oz4x2b5lFbx+vSz2C1QQw+Z5
      JNhQm0IpeFyF7aBJR8Yh3ihIBT61/4QRD02igmECgYEAlKHDjiFB1dcM+Ac456hS
      /UV2As3ZxVaP7qzU63ESk8Gpl9U1zwsVyLOa3/wquYvZ9lTtpLk1LO22EhepgFRv
      59mt1cjlYB+tJ9gh2MYvTA4vh8ooebgS+k5muII3eoo646gtbWtth5ohqIQh5u3b
      OCt3xtlfdxPkal6M3L09ovw=
      -----END RSA PRIVATE KEY-----

    public_key: |
      -----BEGIN PUBLIC KEY-----
      MIIBIjANBgkqhkiG9w0BAQEFAAOCAQ8AMIIBCgKCAQEAq7KL6FyF7DLYBChmwh/u
      ivcbaA+GMumhph47Pt9s4iHKaZT15LyP8LyFOw9W5gjZUv9h9UfxdFWCpG2Vmf/9
      8agSgDUuMBqk60gvvv9ByA6xPdxKRDURHTxQ8Cmr8FXao+T3cZ1FmIz4SO6nbTPj
      vGRC8poxAw4VXAzaPCuWruXlj/8WCxxPpUaS4c380He/RvsRvoaoJ/i57S8sIflZ
      LMvq0NGfgr48BQkGw7rdDByWF8azupwDAoZijtxl5rzm9fAQBFp4V4whW+1YAvCa
      DVMjbYwcYmv4QClRTI2MeC34isyHMc4ZKCfATGjco4fhC+YRQm/T/ietNZiVrsu8
      cQIDAQAB
      -----END PUBLIC KEY-----

msi:
  output_path: "./dist/cx-win/cx.msi"
  product_name: "CX"

