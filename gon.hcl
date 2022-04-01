# gon.hcl
#
# The path follows a pattern
# ./dist/BUILD-ID_TARGET/BINARY-NAME
source = ["./bin/cx-mac"]
bundle_id = "com.checkmarx.cli"

apple_id {
  username = "tiago.baptista@checkmarx.com"
  password = "@env:AC_PASSWORD"
}

sign {
  application_identity = "iPhone Distribution: CHECKMARX LTD (Z68SAQG5BR)"
}

dmg {
  output_path = "cx-signed.dmg"
  volume_name = "cx-signed"
}
