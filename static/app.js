Vue.component('file-component', { 
    props: [
        "file" // the raw File type
    ],
    computed: {
        humanSize() {
            return humanFileSize(this.file.size, true)
        },
        // Use the full path or just the name if its empty
        name() {
            if (this.file.webkitRelativePath) {
                return this.file.webkitRelativePath
            }
            return this.file.name
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
    <small>Queued</small>
    </span>`
})


var app = new Vue({
    el: '#app',
    data: {
        scpCommand: 'scp -r scp.click:HvL5Q8qmg .',
        files: []
    },
    methods: {
        appendFiles(refName) {
            // it seems what we are dealing with here is not really "arrays"
            // so we must loop them to add them into this.files
            var len = this.$refs[refName].files.length
            var i = 0
            
            while ( i < len ) {
                // localize file var in the loop
                var file = this.$refs[refName].files[i]
                file.id = uniqueID()
                this.files.push(file)
                i++
            }    
        }
    }
})

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

function uniqueID() {
    return '_' + Math.random().toString(36).substr(2, 9)
};