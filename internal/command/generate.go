package command

//go:generate mockgen -source=runner.go -destination=mock_runner.go -package=command
//go:generate mockgen -source=git.go -destination=mock_git.go -package=command
//go:generate mockgen -source=gh.go -destination=mock_gh.go -package=command
