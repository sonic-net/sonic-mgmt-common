/*
 * Copyright (c) 2016 Dell Inc.
 *
 * Licensed under the Apache License, Version 2.0 (the "License"); you may
 * not use this file except in compliance with the License. You may obtain
 * a copy of the License at http://www.apache.org/licenses/LICENSE-2.0
 *
 * THIS CODE IS PROVIDED ON AN  *AS IS* BASIS, WITHOUT WARRANTIES OR
 * CONDITIONS OF ANY KIND, EITHER EXPRESS OR IMPLIED, INCLUDING WITHOUT
 *  LIMITATION ANY IMPLIED WARRANTIES OR CONDITIONS OF TITLE, FITNESS
 * FOR A PARTICULAR PURPOSE, MERCHANTABLITY OR NON-INFRINGEMENT.
 *
 * See the Apache Version 2.0 License for specific language governing
 * permissions and limitations under the License.
 */

/*
 * filename: sdi_entity_framework.c
 */


/**************************************************************************************
 * Core SDI framework which provides core api that work on entity.
 ***************************************************************************************/

#include "sdi_entity.h"


static const char empty_string[] = "";

/**
 * Returns the name of entity
 */
const char *sdi_entity_name_get(sdi_entity_hdl_t hdl)
{
    return (empty_string);      /* Valid, but dummy value */
}

/**
 * Returns the type of entity
 */
sdi_entity_type_t sdi_entity_type_get(sdi_entity_hdl_t hdl)
{
    return (SDI_ENTITY_SYSTEM_BOARD);  /* Valid, but dummy value */
}

/**
 * Retrieve number of entities supported by system of given type
 * etype[in] : entity type
 * return number of entities of the specified type
 */
uint_t sdi_entity_count_get(sdi_entity_type_t etype)
{
    return (0);                 /* Valid, but dummy value */
}

/**
 * Iterate on entity list and run specified function on every entity
 *
 * hdl[in] - entity handle
 * fn[in] - function that would be called for each entity
 * user_data[in] - user data that will be passed to the function
 */
void sdi_entity_for_each(void (*fn)(sdi_entity_hdl_t hdl, void *user_data),
                         void *user_data)
{
}

/**
 * Retrieve the handle of the specified entity.
 * etype[in] - Type of entity
 * instance[in] - Instance of the entity of specified type that has
 * to be retrieved.
 * return - If the API succeeds, the handle to the specified entity,
 * else NULL.
 */
sdi_entity_hdl_t sdi_entity_lookup(sdi_entity_type_t etype, uint_t instance)
{
    return (0);                 /* Valid, but dummy value */
}

/**
 * Retrieve number of resources of given type within given entity.
 * hdl[in] - handle to the entity whose information has to be retrieved.
 * resource_type[in] - type of resource. Example, temperature, fan etc.
 * return - returns the number of entities of the specified type that
 * are supported on this system.
 */
uint_t sdi_entity_resource_count_get(sdi_entity_hdl_t hdl, sdi_resource_type_t resource_type)
{
    return (0);                 /* Valid, but dummy value */
}

/**
 * Retrieve the handle of the resource whose name is known.
 * hdl[in] - handle to the entity whose information has to be
 * retrieved.
 * resource[in] - The type of resource that needs to be looked up.
 * alias[in] - the name of the alias. example, "BOOT_STATUS" led.
 * return - if a resource maching the criteria is found, returns handle to it.
 * else returns NULL.
 */
sdi_resource_hdl_t sdi_entity_resource_lookup(sdi_entity_hdl_t hdl,
                                              sdi_resource_type_t resource, const char *alias)
{
    return (0);                 /* Valid, but dummy value */
}

/**
 * Retrieve the alias name of the given resource.
 * resource_hdl[in] - handle to the resource whose name has to be retrieved.
 * return  - the alias name of the resource. example, "BOOT_STATUS" led.
 * else returns NULL.
 */
const char * sdi_resource_alias_get(sdi_resource_hdl_t resource_hdl)
{
    return (empty_string);      /* Valid, but dummy value */
}

/**
 * Iterate on each resource and run specified function on every entity
 *
 * hdl[in] - Entity handle
 * fn[in] - function that would be called for each resource
 * user_data[in] - user data that will be passed to the function
 */
void sdi_entity_for_each_resource(sdi_entity_hdl_t hdl,
                                  void (*fn)(sdi_resource_hdl_t hdl, void *user_data),
                                  void *user_data)
{
}

/**
 * Initialize the specified entity.
 * Upon Initialization, default configurations as specified for platform would
 * be applied
 * param[in] hdl - handle to the entity whose information has to be initialised.
 * return STD_ERR_OK on success and standard error on failure
 */
t_std_error sdi_entity_init(sdi_entity_hdl_t hdl)
{
    return (STD_ERR_OK);
}

/**
 * @TODO: Below two functions are added as a interim solution for PAS
 * compilation.These functions will be removed once the PAS team clean-up
 * their code.
 */

/**
 * Retrieve the handle of first resource of the specified type within the entity.
 * hdl[in] -  handle to the entity whose information has to be retrieved.
 * resource[in] - The type of resource that needs to be looked up.
 * return - if a resource maching the criteria is found, returns handle to it.
 *          else returns NULL.
 */
sdi_resource_hdl_t sdi_entity_get_first_resource(sdi_entity_hdl_t hdl,
                                                 sdi_resource_type_t resource)
{
    return (0);                 /* Valid, but dummy value */
}
/**
 * Retrieve the handle of next resource of the specified type within the entity.
 * hdl[in] - handle to the entity whose information has to be retrieved.
 * resource[in] - The type of resource that needs to be looked up.
 * return - if a resource maching the criteria is found, returns handle to it.
 *          else returns NULL.
 */
sdi_resource_hdl_t sdi_entity_get_next_resource(sdi_resource_hdl_t hdl,
                                                sdi_resource_type_t resource)
{
    return (0);                 /* Valid, but dummy value */
}

/**
 * Returns the type of resource from resource handler
 */
sdi_resource_type_t sdi_resource_type_get(sdi_resource_hdl_t hdl)
{
    return (SDI_RESOURCE_FAN);   /* Valid, but dummy value */
}
