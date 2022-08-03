# RCS Native TCP Protocol (RCSP)

## Requests

### SET

```
RCSP/1.0 SET\r\n
KEY: <key>\r\n
VALUE: <val>\r\n  
```

### GET

```
RCSP/1.0 GET\r\n
KEY: <key>\r\n
```

### DELETE

```
RCSP/1.0 DELETE\r\n
KEY: <key>\r\n
```

### PURGE

```
RCSP/1.0 PURGE\r\n
KEY: <key>\r\n
```

### LENGTH

```
RCSP/1.0 LENGTH\r\n
```

### KEYS

```
RCSP/1.0 KEYS\r\n
```

### PING

```
RCSP/1.0 PING\r\n
```

### CLOSE

```
RCSP/1.0 CLOSE\r\n
```

## Responses

### SET OK

```
RCSP/1.0 SET OK\r\n
KEY: <key>\r\n
```

### SET NOT_OK

```
RCSP/1.0 SET NOT_OK\r\n
MESSAGE: <msg>\r\n
KEY: <key>\r\n
```

### GET OK

```
RCSP/1.0 GET OK\r\n
KEY: <key>\r\n
VALUE: <val>\r\n
```

### GET NOT_OK

```
RCSP/1.0 GET NOT_OK\r\n
MESSAGE: <msg>\r\n
KEY: <key>\r\n
```


### DELETE OK

```
RCSP/1.0 DELETE OK\r\n
KEY: <key>\r\n
```

### DELETE NOT_OK

```
RCSP/1.0 DELETE NOT_OK\r\n
MESSAGE: <msg>\r\n
KEY: <key>\r\n
```

### PURGE OK

```
RCSP/1.0 PURGE OK\r\n
```

### PURGE NOT_OK

```
RCSP/1.0 PURGE NOT_OK\r\n
MESSAGE: <msg>\r\n
```

### LENGTH OK

```
RCSP/1.0 LENGTH OK\r\n
VALUE: <val>\r\n
```

### LENGTH NOT_OK

```
RCSP/1.0 LENGTH NOT_OK\r\n
MESSAGE: <msg>\r\n
```

### KEYS OK

```
RCSP/1.0 KEYS OK\r\n
VALUE: <val>\r\n
```

### KEYS NOT_OK

```
RCSP/1.0 KEYS NOT_OK\r\n
MESSAGE: <msg>\r\n
```

### PING OK

```
RCSP/1.0 PING OK\r\n
```

### PING NOT_OK

```
RCSP/1.0 PING NOT_OK\r\n
MESSAGE: <msg>\r\n
```

### CLOSE OK

```
RCSP/1.0 CLOSE OK\r\n
```

### CLOSE NOT_OK

```
RCSP/1.0 CLOSE NOT_OK\r\n
```

### Generic Error Response

```
RCSP/1.0 NOT_OK\r\n
MESSAGE: <msg>\r\n
```
