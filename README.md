Open Data Proxy
===============

Usage:
------

Add a settings.json to the same directory as your executable with the following settings:

json```
{
    "http_timeout": 30,
    "http_port": "80",
    "username": "email",
    "password": "password"
}
```

Installing:
-----------

Dependencies are installed using Glide.

bash```
glide init
make install or glide up
```


Tests
-----

bash```
make tests
```


https://opendata.tweedekamer.nl/system/files/documentation/open_data_portaal_api_beschrijvingen.pdf