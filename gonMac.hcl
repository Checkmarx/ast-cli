# gonArm.hcl
source = ["./dist/cx-mac-universal_darwin_all/cx"]
bundle_id = "com.checkmarx.cli"

apple_id {
  username = "tiago.baptista@checkmarx.com"
  password = "@env:AC_PASSWORD_2024"
  provider = "Z68SAQG5BR"
}

sign {
  application_identity = "Developer ID Application: CHECKMARX LTD"
}

dmg {
  output_path = "./dist/cx-mac-universal_darwin_all/cx.dmg"
  volume_name = "cx"
}
