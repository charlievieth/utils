/* Generated by the protocol buffer compiler.  DO NOT EDIT! */
/* Generated from: request.proto */

#ifndef PROTOBUF_C_request_2eproto__INCLUDED
#define PROTOBUF_C_request_2eproto__INCLUDED

#include <protobuf-c/protobuf-c.h>

PROTOBUF_C__BEGIN_DECLS

#if PROTOBUF_C_VERSION_NUMBER < 1003000
# error This file was generated by a newer version of protoc-c which is incompatible with your libprotobuf-c headers. Please update your headers.
#elif 1003003 < PROTOBUF_C_MIN_COMPILER_VERSION
# error This file was generated by an older version of protoc-c which is incompatible with your libprotobuf-c headers. Please regenerate this file with a newer version of protoc-c.
#endif

typedef struct _Request Request;

/* --- enums --- */

/* --- messages --- */

struct _Request {
	ProtobufCMessage base;
	uint64_t session_id;
	uint32_t ppid;
	int32_t status_code;
	uint64_t history_id;
	char *wd;
	char *username;
	char *command;
	size_t n_args;
	char **args;
};
#define REQUEST__INIT                                                                   \
	{                                                                                   \
		PROTOBUF_C_MESSAGE_INIT(&request__descriptor)                                   \
		, 0, 0, 0, 0, (char *)protobuf_c_empty_string, (char *)protobuf_c_empty_string, \
		    (char *)protobuf_c_empty_string, 0, NULL                                    \
	}

/* Request methods */
void request__init(Request *message);
size_t request__get_packed_size(const Request *message);
size_t request__pack(const Request *message, uint8_t *out);
size_t request__pack_to_buffer(const Request *message, ProtobufCBuffer *buffer);
Request *request__unpack(ProtobufCAllocator *allocator, size_t len, const uint8_t *data);
void request__free_unpacked(Request *message, ProtobufCAllocator *allocator);
/* --- per-message closures --- */

typedef void (*Request_Closure)(const Request *message, void *closure_data);

/* --- services --- */

/* --- descriptors --- */

extern const ProtobufCMessageDescriptor request__descriptor;

PROTOBUF_C__END_DECLS

#endif /* PROTOBUF_C_request_2eproto__INCLUDED */