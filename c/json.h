
// ॐ //

#pragma once

#include "ujdecode.h"
#include "buf.h"

void parse_string(UJObject str);
void parse_int(UJObject i);
void parse_user(UJObject user);
void parse_json(buf_t* buf);
