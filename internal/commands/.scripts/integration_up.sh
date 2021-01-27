echo "Authenticating to AST test server."
#./ast.exe auth register -u XXX -p XXX > auth.txt
# export $(awk 'NR==1 { print }' auth.txt | awk -F'=' '{ print "export AST_ACCESS_KEY_ID==" $2 }')
# export $(awk 'NR==2 { print }' auth.txt | awk -F'=' '{ print "AST_ACCESS_KEY_SECRET==" $2 }')
# export BASE_URI="http://ast.cxflow-ci.com"
# export BASE_URI="http://127.0.0.1:8080"
# export AST_ACCESS_KEY_ID="ast-plugins-3d1b8499-b3cf-43cf-8425-0bb212ca44d3"
# export AST_ACCESS_KEY_SECRET="c710a92f-d103-4bc5-89ce-6d138f976b9e"
echo "Kicking off integration tests."
go test ./test/integration/scan_test.go \
            ./test/integration/root_test.go \
            ./test/integration/project_test.go \
            ./test/integration/health_check_test.go \
            ./test/integration/result_test.go \
            ./test/integration/query_test.go \
            ./test/integration/sast_resources_test.go -v
echo "Done running integration tests."