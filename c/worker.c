
// ॐ //

#include <curl/curl.h>
#include <unistd.h>

#include "buf.h"
#include "json.h"
#include "worker.h"

char* url = "https://api.vk.com/method/"
            "users.get?v=3&fields=bdate,city,country,sex&uids=1,2";

static size_t on_chunk(void* contents, size_t size, size_t nmemb, void* userp) {
    size_t realsize = size * nmemb;
    buf_t* buf      = (buf_t*)userp;
    Buf_write(buf, contents, realsize);
    return realsize;
}

void* worker(void* tid) {

    CURL*    curl;
    CURLcode res;

    curl_global_init(CURL_GLOBAL_ALL);
    curl = curl_easy_init();
    if (curl) {
        /*
        curl_easy_setopt(curl, CURLOPT_STDERR, stdout);
        curl_easy_setopt(curl, CURLOPT_VERBOSE, 1L);
        curl_easy_setopt(curl, CURLOPT_HEADER, 1L);
        */
        buf_t* buf = Buf_new();
        curl_easy_setopt(curl, CURLOPT_WRITEFUNCTION, on_chunk);
        curl_easy_setopt(curl, CURLOPT_WRITEDATA, buf);
        for (int i = 0; i < 3; i++) {
            curl_easy_setopt(curl, CURLOPT_URL, url);
            res = curl_easy_perform(curl);
            // process buffer
            parse_json(buf);
            // cleanup
            Buf_empty(buf);
            if (res != CURLE_OK)
                fprintf(stderr, curl_easy_strerror(res));
        }
        Buf_free(buf);
        curl_easy_cleanup(curl);
    }
    return NULL;
}
