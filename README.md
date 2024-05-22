# Revel module to operate git

# Usage:

## Installation

Install module

```
# specific version
go get go get github.com/QFO6/rev-git@vx.x.x
# or get latest
go get github.com/QFO6/rev-git@master
```

## Setup

Include module in app.conf

```
module.revgit=github.com/QFO6/rev-git
```

Include module in conf/routes

```
module:revgit
```

Needs to define routes in under your revel_app/conf/routes file

```
GET    /api/git/:modelName/:id/commit/:commitHash                          GitAPI.CommitContent
POST   /api/git/:modelName/:id/commit                                      GitAPI.Commit
GET    /api/git/:modelName/:id/history                                     GitAPI.History
```

Init Git config before call the apis

```
func initRevGit(session *mgo.Session) revel.Result {
	utilData := new(revmongo.Utils)
	do := revmongo.New(session, utilData)
	do.Query = bson.M{"Name": revgit.GitUtilName}
	do.GetByQ()
	if !utilData.Id.Valid() {
		fmt.Printf("No valid %s util configured\n", revgit.GitUtilName)
		return nil
	}

	if utilData.Value == "" {
		fmt.Printf("No valid %s util value configured\n", revgit.GitUtilName)
		return nil
	}

	revgit.Init(utilData)
	return nil
}

// init Git
session := revmongo.NewMgoSession()
initRevGit(session)
```

Re-init Git config after change the GitConfig util from UI side
Fex.

```
if newUtilData.Name == revgit.GitUtilName {
  revgit.Init(newUtilData)
}
```

### Note:

Add a util with named 'GitConfig' in your application utils page with following json string format:

```
{
  "grpcUrl": "test.abc.com:8051",
  "gitUrl": "https://git.abc.com/<org_name>/<repo_name>.git",
  "gitToken": "xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx"
}
```

