Vue.component('file-component', { 
    props: [
        "file" // Our definition of a file
    ],
    computed: {
        humanSize() {
            return humanFileSize(this.file.raw.size, true)
        },
        // Use the full path or just the name if its empty
        name() {
            if (this.file.raw.webkitRelativePath) {
                return this.file.raw.webkitRelativePath
            }
            return this.file.raw.name
        }
    },
    template: `<span href="#" class="list-group-item list-group-item-action flex-column align-items-start">
    <div class="d-flex w-100 justify-content-between">
        <h5 class="mb-1">{{ name }}</h5>
        <small> 0 / {{ humanSize }}</small>
    </div>
    <div class="progress">
        <div class="progress-bar" style="width: 0%;" role="progressbar" aria-valuenow="0" aria-valuemin="0" aria-valuemax="100"></div>
    </div>
    <small>{{ file.state }}</small>
    </span>`
})


var app = new Vue({
    el: '#app',
    data: {
        scpCommand: 'scp -r scp.click:HvL5Q8qmg .',
        files: [],
        ws: undefined,
        nextId: 1,
        transferReady: true,
    },
    methods: {
        appendFiles(refName) {
            // it seems what we are dealing with here is not really "arrays"
            // so we must loop them to add them into this.files
            var len = this.$refs[refName].files.length
            var i = 0
            
            while ( i < len ) {
                // localize file var in the loop

                var file = NewFile(this.nextId, this.$refs[refName].files[i])
                this.nextId++
                
                this.files.unshift(file)
                i++
            }

            if (this.transferReady) {
                this.transmitNextFile()
            }    
        },
        // transmitNextFile reads files from the top of this.files
        // and transmit's it - then i calls it self again
        transmitNextFile() {
            var file = this.files[0];
            if (file.state != "queued") {
                return
            }
            var reader = new FileReader();

            reader.onload = function(e) {

                
                this.ws.send(e.target.result)
                console.log("the File has been transferred.")

                file.state = "transfered"
                
                // put this top file at the bottom of the array
                this.files.push(this.files.shift())
                
                this.transmitNextFile()
            }.bind(this)

            

            reader.onprogress = (ev) => {
                console.log("Progress:", ev)
            }

            reader.onerror = (error) => {
                console.log("filereader error:", error)
            }

            reader.readAsArrayBuffer(file.raw);
        }
    },
    mounted() {

        // WebSocket wants absolute urls
        // figure out if we are on http(s) and what our hostname is
        var loc = window.location, new_uri;
        if (loc.protocol === "https:") {
            new_uri = "wss:";
        } else {
            new_uri = "ws:";
        }
        new_uri += "//" + loc.host;
        new_uri += "/source/";
        
        ws = new WebSocket(new_uri);
        ws.binaryType = "arraybuffer";

        ws.onopen = function(evt) {
            console.log("OPEN");
            ws.send("hello")
        }
        
        ws.onclose = function(evt) {
            console.log("CLOSE");
            ws = null;
        }
        
        ws.onmessage = function(evt) {
            console.log("RESPONSE: ", evt.data);
        }

        ws.onerror = function(evt) {
            console.log("ERROR: ", evt.data);
        }

        this.ws = ws
    }
})

function NewFile(id, file) {
    return {
        id: id,
        raw: file,
        state: "queued",
    }
}

// special thanks to `mpen` from https://stackoverflow.com/questions/10420352/converting-file-size-in-bytes-to-human-readable-string
function humanFileSize(bytes, si) {
    var thresh = si ? 1000 : 1024;
    if(Math.abs(bytes) < thresh) {
        return bytes + ' B';
    }
    var units = si
        ? ['kB','MB','GB','TB','PB','EB','ZB','YB']
        : ['KiB','MiB','GiB','TiB','PiB','EiB','ZiB','YiB'];
    var u = -1;
    do {
        bytes /= thresh;
        ++u;
    } while(Math.abs(bytes) >= thresh && u < units.length - 1);
    return bytes.toFixed(1)+' '+units[u];
}
