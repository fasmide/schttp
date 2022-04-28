```
     ___  ___ _ __  
    / __|/ __| '_ \ 
    \__ \ (__| |_) |
    |___/\___| .__/ 
            | |    
            |_|    
             _ _      _    
            | (_)    | |   
         ___| |_  ___| | __
        / __| | |/ __| |/ /
       | (__| | | (__|   < 
        \___|_|_|\___|_|\_\
        (click, not dick)
```
# What is it
schttp is an ssh daemon that makes filesharing between boxes and people easier. 

# How it works
With your regular SCP client you can transfer some files or directories to schttp which on-the-fly zip or tgz compresses the stream and provides you with a URL to share with your peers. 

This way, you can quickly get some files off a box directly from the command line, without having to do a key exchange or anything with those accepting the data.

# Try it out

```
$ scp -r some-directory scp.click: 
```

Nothing happens until a peer begins to downloads the url :)