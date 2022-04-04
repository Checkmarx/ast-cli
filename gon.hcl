# gon.hcl
#
# The path follows a pattern
# ./dist/BUILD-ID_TARGET/BINARY-NAME
source = ["./dist/cx-mac_darwin_arm64/cx"]
bundle_id = "com.checkmarx.cli"

apple_id {
  username = "tiago.baptista@checkmarx.com"
  password = "@env:AC_PASSWORD"
}

sign {
  application_identity = "Mac Developer: Galactica Team (M743PYYBKU)"
}

dmg {
  output_path = "cx-signed.dmg"
  volume_name = "cx-signed"
}
