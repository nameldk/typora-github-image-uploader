An image upload tool that uploads the image to GitHub works for Typora.

### usage

download project

create config.json

```json
{
  "repo": "owner/projectName",
  "branch": "main",
  "token": "github access token",
  "path": "image/2013"
}
```

    go build .

`config Typora:`

    Typora Preferences \ image \ image upload setting \ image uploader \ custom command

    command: path/to/typora-github-image-uploader.exe
