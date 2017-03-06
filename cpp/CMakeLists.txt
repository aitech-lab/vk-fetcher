add_executable(users.get users.get.cpp )
link_directories(/usr/local/lib)
target_link_libraries(users.get PocoNetSSL PocoCrypto PocoNet PocoUtil PocoJSON PocoXML PocoFoundation )
