Vue.component('file-component', { 
    props: [
        "file" // Our definition of a file
    ],
    computed: {
        humanSize() {
            return humanFileSize(this.file.raw.size, true)
        },
        progressStyle() {
            return "width: " + this.file.progress + "%;"
        },
        progressState() {
            if (this.file.state == "transfered") {
                return {
                    "bg-success": true
                }
            }
            if (this.file.state == "failed") {
                return {
                    "bg-danger": true
                }
            }
            return {}
        },
        humanLoadedSize() {
            return humanFileSize(this.file.loaded, true)
        },
        status() {
            switch (this.file.state) {
                case "queued":
                    return "In Queue"
                case "transfering":
                    return "Uploading: " + humanFileSize(this.file.rate, true) + "/sec"
                case "transfered":
                    var delta = (this.file.finishedAt - this.file.startedAt) / 1000
                    var rate = this.file.raw.size/delta
                    return "Completed in " + delta + " seconds at " + humanFileSize(rate, true) + "/sec"
                case "failed":
                    return this.file.message
                default:
                    console.log("unknown file state:", this.file.state)
                    return "N/A"
            }
        }
    },
    template: `<span href="#" class="list-group-item flex-column align-items-start">
    <h5 class="mb-1 text-truncate">{{ file.path }}</h5>
    <div class="progress"><div class="progress-bar" :class="progressState" :style="progressStyle" role="progressbar"></div></div>
    <span class="d-flex justify-content-between">
        <small>{{ status }}</small>
        <small> {{ humanLoadedSize }} / {{ humanSize }}</small>
    </span>
</span>`
})


var app = new Vue({
    el: '#app',
    data: {
        scpCommand: 'scp -r scp.click:HvL5Q8qmg .',
        files: [],
        nextId: 1,
        transferReady: true,
    },
    methods: {
        highlightDrop(state) {
            if (state) {
                this.$refs["drophighlight"].classList.remove("d-none")    
                return
            }
            this.$refs["drophighlight"].classList.add("d-none")
        },
        async handleDrop(e) {
            // prevent the browsers default behavior
            // otherwise it will try to open the file dropped it self
            e.stopPropagation()
            e.preventDefault()

            // hide the dropzone
            this.highlightDrop(false)

            return new Promise(async (resolve) => {   
                var items = event.dataTransfer.items
                for (var i=0; i<items.length; i++) {
                    // webkitGetAsEntry is where the magic happens
                    var item = items[i].webkitGetAsEntry()
                    if (item) {
                        await this.traverseItem(item)
                    }
                }

                // transmit files if we are ready
                if (this.transferReady) {
                    this.transmitNextFile()
                } 
                resolve()
            })
        },
        // TODO: instead of adding every single file to this.files
        // we should be able to handle directories as a single unit
        async traverseItem(item, path) {
            return new Promise(async (resolve) => {
                path = path || ""
                
                if (item.isFile) {
                    // Get file
                    item.file((file) => {
                        
                        var f = NewFile(this.nextId, file)
                        f.path = path + file.name;
                        this.nextId++
                        
                        this.files.unshift(f)
                        resolve()
                    });
                    return
                }

                if (item.isDirectory) {
                    // Get folder contents
                    var dirReader = item.createReader();
                    dirReader.readEntries(async (entries) => {
                        for (var i=0; i<entries.length; i++) {
                            await this.traverseItem(entries[i], path + item.name + "/");
                        }
                        resolve()
                    });
                    return
                }

                // so it was not a file, and it was no directory....
                console.log("Found something that was no file or directory", item)
                resolve()
                
            })
        },
        appendFiles(refName) {
            // it seems what we are dealing with here is not really "arrays"
            // so we must loop them to add them into this.files
            var len = this.$refs[refName].files.length
            var i = 0
            
            while ( i < len ) {
                // localize file var in the loop

                var file = NewFile(this.nextId, this.$refs[refName].files[i])
                file.path = file.raw.webkitRelativePath || file.raw.name
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

            // are there any files?
            if (this.files.length <= 0) {
                return
            }

            // should we work on this file?
            var file = this.files[0];
            if (file.state != "queued") {
                return
            }

            // initialize new POST request
            var xhr = new XMLHttpRequest();
            xhr.open('POST', '/source/', true);

            // onload fires when the file have been uploaded
            // TODO: this should have some error checking i guess
            xhr.onload = () => {
                if (xhr.status != 200) {
                    file.state = "failed"
                    file.message = xhr.status + ": " + xhr.response
                    return
                }    

                file.state = "transfered"
                file.finishedAt = Date.now()

                // put this top file at the bottom of the array
                this.files.push(this.files.shift())
                
                // advance to the next file
                this.transmitNextFile()
            }

            // these are only network errors and misuse of xhrhttprequest
            // e.g. with in valid urls - i dont know any way to read the 
            // error message
            xhr.onerror = () => {
                file.state = "failed"
                file.message = "Communication error"

                // should we advance to the next file? not quite sure
            }
            
            var lastUpdate = 0
            xhr.upload.onprogress = (e) => {
                // if lastUpdate is zero this is the first update we have
                // we must use startedAt instead of lastUpdate
                var deltaTime = 0
                if (lastUpdate == 0) {
                    deltaTime = Date.now() - file.startedAt
                } else {
                    deltaTime = Date.now() - lastUpdate
                }

                var deltaRate = e.loaded - file.loaded
                file.rate = deltaRate * (1000 / deltaTime)

                file.loaded = e.loaded
                lastUpdate = Date.now()
                
                if (e.lengthComputable) {
                    file.progress = (e.loaded / e.total) * 100
                }
            }

            // this modern xhr should understand the native File type
            file.state = "transfering"
            file.startedAt = Date.now()
            xhr.send(file.raw);

        }
    }
})

function NewFile(id, file) {
    return {
        // id is used by vue to optimize the view somehow
        id: id,

        // raw is the browser's native File type
        raw: file,

        // path to the file
        path: "",

        // state holds queued, transfering, transfered or failed
        state: "queued",

        // progress in percentage
        progress: 0,

        // rate in bytes pr second
        rate: 0,

        // how many bytes are currently sent
        loaded: 0,

        // started and finished times
        startedAt: 0,
        finishedAt: 0,

        // If schttp returns an non 200 code - here will be a message
        message: "",
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
