# Install Chain Core Developer Edition

Chain Core is enterprise-grade software used to operate or participate in a blockchain...

## MacOS/OSX
### [OPTION 1] Download Mac app
[chain-core-1.0-app.zip](#)
### [OPTION 2]Install with Docker

#### Run the latest Docker image
```
$ docker run -it chain:latest -p 8080:8080
```

#### Visit the dashboard
```
$ open http://localhost:8080
```

## Windows
### [OPTION 1] Download Windows installer
[chain-core-1.0-msi.zip](#)

### [OPTION 2] Install with Docker
#### Run the latest Docker image
```
$ docker run -it chain:latest -p 8080:8080
```

#### Visit the dashboard
```
$ open http://localhost:8080
```

#### Confirgure port forwarding (Windows 7, 8, 8.1)
Systems running Windows 7, 8, or 8.1 are unable to run the latest version of Docker (Docker for Windows). Instead, these systems run Docker Toolbox, which utilizes VirtualBox. Therefore, in order to access the Chain Core container through localhost, you must first configure VirtualBox port forwarding to properly route the requests.

If you are running the latest version of Docker on Windows 10, you can skip this step.

![](../images/virtualbox-config.gif)


## Linux
### [ONLY OPTION] Install with Docker

#### Run the latest Docker image
```
$ docker run -it chain:latest -p 8080:8080
```

#### Visit the dashboard
```
$ open http://localhost:8080
```
