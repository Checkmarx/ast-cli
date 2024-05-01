source = ["./dist/cx-win/cx.exe"]
bundle_id = "com.checkmarx.cli"

sign:
  private_key: |
      -----BEGIN RSA PRIVATE KEY-----

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

