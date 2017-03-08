include_directories(ujson4c/src ujson4c/3rdparty)
set(JSON_SRC ujson4c/src/ujdecode.c ujson4c/3rdparty/ultrajsondec.c)
add_executable(users.get users.get.c worker.c buf.c json.c ${JSON_SRC})
target_link_libraries(users.get curl pthread m) #
