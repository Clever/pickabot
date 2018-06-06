package mocks

// generate mocks of dependencies for use during testing
//go:generate sh -c "$PWD/bin/mockgen -package mocks -source $PWD/who_is_who_client_iface.go whoIsWhoClientIface > mock_who_is_who_client_.go"
//go:generate sh -c "$PWD/bin/mockgen -package mocks -source $PWD/slackapi/SlackService.go SlackAPIService,SlackRTMService > mock_slack_service.go"
//go:generate sh -c "$PWD/bin/mockgen -package mocks -source $PWD/github/client.go AppClientIface > mock_github.go"
