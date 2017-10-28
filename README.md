# Open Data Tweede Kamer Proxy

To use the Tweede Kamer Open Data API, you need to whitelist your IP on [Open Data Portaal](https://opendata.tweedekamer.nl/).
Only the white listed IP is able to access the API.
This proxy helps developers to access the API rom anywhere through their whitelisted IP.

## Usage

1. Add a settings.json to the same directory as your executable with the following settings:

```json
{
    "http_timeout": 30,
    "http_port": "80",
    "username": "email",
    "password": "password"
}
```

2. Compile for your specific platform. Linux AMD64 and OSX are included in the makefile.

```bash
make osx
make linux64
```

3. Run the executable

```bash
./build/od2kproxy
```

## Installing:

Dependencies are installed using Glide.

```bash
glide init
make install or glide up
```


## Tests

```bash
make tests
```

## References

[Open Data Tweedekamer API documentation](https://opendata.tweedekamer.nl/system/files/documentation/open_data_portaal_api_beschrijvingen.pdf)
