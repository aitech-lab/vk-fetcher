
// ॐ //

#include <curl/curl.h>
#include <math.h>
#include <string.h>
#include <unistd.h>

#include "buf.h"
#include "json.h"
#include "worker.h"

#define API                                                                    \
    "https://api.vk.com/method/"                                               \
    "users.get?v=3&fields=bdate,city,country,sex&uids="
#define API_LEN sizeof API

const int  max_cnt         = 800;
const int  max_len         = 4000;
const long uids_per_thread = 100000;

static unsigned long generate_url(char* url, unsigned long from) {
    sprintf(url, "%s", API);
    int len  = API_LEN - 1;
    int cnt  = 0;
    int nlen = 0;
    while (cnt++ < max_cnt && len < max_len) {
        nlen = (int)floor(log10(from)) + 2;
        sprintf(url + len, "%d,", from++);
        len += nlen;
    }
    return from;
}

static size_t on_chunk(void* contents, size_t size, size_t nmemb, void* userp) {
    size_t realsize = size * nmemb;
    buf_t* buf      = (buf_t*)userp;
    buf_write(buf, contents, realsize);
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

        buf_t* buf = buf_new();
        curl_easy_setopt(curl, CURLOPT_WRITEFUNCTION, on_chunk);
        curl_easy_setopt(curl, CURLOPT_WRITEDATA, buf);

        char url[max_len + 100];

        unsigned long from = ((long)tid) * uids_per_thread+1;
        unsigned long to   = from + uids_per_thread+1;

        while (from < to) {
            from = generate_url(url, from);
            curl_easy_setopt(curl, CURLOPT_URL, url);

            res = curl_easy_perform(curl);
            // process buffer
            parse_json(buf);
            // cleanup
            buf_empty(buf);
            if (res != CURLE_OK)
                fprintf(stderr, curl_easy_strerror(res));
        }
        buf_free(buf);
        curl_easy_cleanup(curl);
    }
    return NULL;
}
