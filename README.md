# Influsender
Small utility to send main metrics (load average, uptime, mem, disk, io, net) to the influx database

## Pros

* It uses only syscall and reading files from /proc
* It doesn't need root privileges
* It hasn't a lot of dependencies
* It's crossplatform, binaries that I've made with ./build.sh working fine on arm and arm64 platforms like Raspberry/Orange/BananaPi and others
* One run is one query to database
* You can switch off metrics that you don't need or append mountpoints or interfaces to monitoring
* It's simple to change code if you need
* In common case you doesn't need to change ini file and it can be run without it (if you using database with name influsender)
* It's rather small (about 7Mb without UPX compression and about 4Mb with)

## Cons

* It's my first experience with golang and made in two evernings so code can be ugly but it works :)
* It's not so small as I expected
* It hasn't ability of run external scripts or commands for extending functional (but I think about make it)
* It can't run from docker (by oblibious reasons)
* For running under musl-based distributives like an Alpine linux it should be builded on them (tested with Alpine linux v3.13)

## Usage:

At first step make database into influxdb:

```
docker exec -it influxdb_container_name influx
create database influsender;
```

Then clone repository, enter into directory with code copy ini file from example (if you need), install golang and make the binary:

```
git clone https://github.com/alive-corpse/influsender
cd influsender
cp influsender.ini_example influsender.ini
sudo apt install golang
go build influsender.go
```

Change influsender.ini if you need (you can get some help messages from ini file's comments), append influsender.sh into cron job.

Also you can build it for various platforms by the ./build.sh running. If you've installed upx package build.sh will make also compressed binaries. They are about twice smaller, but takes more significantly more time to run (difference also depends on platform you've run it on).

#### Feel free to contact me by email: evgeniy.shumilov at gmail.com
