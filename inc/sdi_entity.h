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
 * filename: sdi_entity.h
 * (c) Copyright 2014-2015 Dell Inc. All Rights Reserved.
 */



/**
 * @file sdi_entity.h
 * @brief SDI Public API.
 *
 * @todo Add API for dealing with entity info
 */

#ifndef __SDI_ENTITY_H_
#define __SDI_ENTITY_H_

#include "std_error_codes.h"
#include "std_type_defs.h"

#ifdef __cplusplus
extern "C" {
#endif

/**
 * @defgroup sdi_sys SDI System API.
 * SDI views the system as a set of entities  where each entity has one or
 * more resources.
 * refer to @ref entity_api and @ref sdi_resource for details about entities and
 * resources respectively.
 *
 * @{
 *
 */

/**
 * Different types of entities supported by SDI
 * @note that not all entities are supported on all platforms.
 * Refer to platform documentation to idenitify what entities are
 * supported for a given platform.
 */
typedef enum {
    /** SDI_ENTITY_SYSTEM_BOARD also known as base-board.  This is board
     * on which typically system CPU resides.
     */
    SDI_ENTITY_SYSTEM_BOARD,
    /** SDI_ENTITY_FAN_TRAY identifies the entity as fantray. */
    SDI_ENTITY_FAN_TRAY,
    /** SDI_ENTITY_PSU_TRAY identifies the entity as PSU-tray. */
    SDI_ENTITY_PSU_TRAY,
} sdi_entity_type_t;

/**
 * @defgroup sdi_entity_reset_types SDI ENTITY RESET TYPES.
 * List of entity reset types supported by SDI.
 *
 * @ingroup sdi_sys
 * @{
 */
typedef enum {
    WARM_RESET,  /**<Resets all components of the entity except the Data-plane */

    COLD_RESET, /**<Resets all the components of the entity which includes Control
                  * plane and Data Plane. */
    MAX_NUM_RESET, /**<End of Reset type. Should be at the END */
} sdi_reset_type_t;
/**
 * @}
 */

/** @brief Resource is the smallest element of sdi that can be manipulated.
 */
typedef enum {
    /** Resource monitors temperature.
      * see also @ref sdi_temperature_api */
    SDI_RESOURCE_TEMPERATURE,
    /** Resource monitors temperature.
      * @ref sdi_fan_api */
    SDI_RESOURCE_FAN,
    /** Resource monitors/control fan.
      * @ref sdi_led_api */
    SDI_RESOURCE_LED,
    /** Digital LED, used to display numbers  */
    SDI_RESOURCE_DIGIT_DISPLAY_LED,
    /** Resource that holds entity information */
    SDI_RESOURCE_ENTITY_INFO,
    /** Resource to deal with devices that could be upgraded by SW.
      * @ref sdi_pld_upgrade_api */
    SDI_RESOURCE_UPGRADABLE_PLD,
    /** Resource that deals with Media(sfp/qsfp)
      * @ref sdi_media_api **/
    SDI_RESOURCE_MEDIA
} sdi_resource_type_t;

/**
 * An opaque data-type to identify an resource
 */
typedef void *sdi_hdl_t;

/** @brief Control thresholds/limits for various SDI resource.
  * @note not all thresholds/limits are supported by all resources.
  */
typedef enum {
    SDI_LOW_THRESHOLD,
    SDI_HIGH_THRESHOLD,
    SDI_CRITICAL_THRESHOLD,
} sdi_threshold_t;

/**
 * @brief Initialize the SDI subsystem.
 * The API loads the configuration and initializes SDI subsystem.
 * This API must be called before invoking any other API.
 * @returns stand error message.
 */
t_std_error sdi_sys_init(void);

/**
 * @}
 */

/**
 * @defgroup entity_api SDI Entity API
 * These API deal with operations on an entity like presence, fault,
 * entity_information etc.
 *
 * Entity : An entity is one which the user is able to co-relate to. Hence it
 * can in other words be called a "entity_info" device. An entity would consist
 * of devices.
 *
 * @note
 * An entity is identified by it's type and instance, where instance is an 0 based index.
 * Thus the first fan tray is identified by the tuple {SDI_ENTITY_FAN_TRAY, 0}, the second
 * fan tray by {SDI_ENTITY_FAN_TRAY, 1} and so on.
 *
 * @ingroup sdi_sys
 * @{
 */

/**
 * An opaque handle to an entity.  This handle has to be first obtained using
 * sdi_entity_lookup function.
 * All API that work on entity will use this handle as first parameter.
 */
typedef sdi_hdl_t sdi_entity_hdl_t;

/**
 * An opaque handle to an resource.  This handle has to be first obtained using
 * sdi_entity_resource_lookup function.
 */
typedef sdi_hdl_t sdi_resource_hdl_t;

/**
 * @brief  Retrieve number of entities supported by system of given type.
 *    Example to query how many fantrays are supported.
 * @param[in] etype - The type of entity. Example, fantray, psu_tray etc..
 * @return - number of entities of the specified type that can be supported.
 */
uint_t sdi_entity_count_get(sdi_entity_type_t etype);


/**
 * @brief retrieve the handle of the specified entity.
 * @param[in] etype - Type of entity
 * @param[in] instance - Instance of the entity of specified type that has
 *                  to be retrieved.
 * @return - If the API succeeds, the handle to the specified entity, else NULL.
 */
sdi_entity_hdl_t sdi_entity_lookup(sdi_entity_type_t etype, uint_t instance);

/**
 * @brief retrive the type of the entity.
 * @param[in] hdl handle of the entity whose name has to be found.
 * @return type of entity.
 */
sdi_entity_type_t sdi_entity_type_get(sdi_entity_hdl_t hdl);

/**
 * @brief retrive the name of the entity.
 * @param[in] hdl - handle of the entity whose name has to be found.
 * @return name of the entity.
 */
const char *sdi_entity_name_get(sdi_entity_hdl_t hdl);

/**
 * @brief - apply the specified function for every entity in the system.
 * for every entity in the sytem, user-specified function "fn" will be called with
 * corresponding entity and user-specified data ar arguments.
 *
 * @param[in] fn - The function that would be called for each entity in the system.
 * @param[in] user_data - User data that will be passed to the function
 */
void sdi_entity_for_each(void (*fn)(sdi_entity_hdl_t, void *user_data),
        void *user_data);

/**
 * @brief  Reset the specified entity.
 * Reset of entity results in reset of resources and devices as per the reset type.
 * see also @ref sdi_entity_reset_types
 * Upon reset, default configurations as specified for platform would be applied.
 * @param[in] hdl - handle to the entity whose information has to be retrieved.
 * @param[in] type – type of reset to perform.
 * @return - @ref t_std_error
 */
t_std_error sdi_entity_reset(sdi_entity_hdl_t hdl, sdi_reset_type_t type);

/**
 * @brief  Change/Control the power status for the specified entity.
 *
 * @param[in] hdl - handle to the entity whose information has to be retrieved.
 * @param[in] enable – power state to enable / disable
 * @return - @ref t_std_error
 */
t_std_error sdi_entity_power_status_control(sdi_entity_hdl_t hdl, bool enable);

/**
 * @brief:  Initialize the specified entity.
 * Upon Initialization, default configurations as specified for platform would
 * be applied
 * @param[in] hdl - handle to the entity whose information has to be retrieved.
 * @return - @ref t_std_error
 */
t_std_error sdi_entity_init(sdi_entity_hdl_t hdl);

/**
 * @brief  Retrieve presence status of given entity.
 * @param[in] hdl - handle to the entity whose information has to be retrieved.
 * @param[out] pres - true if entity is present, false otherwise
 * @return - @ref t_std_error
 */
t_std_error sdi_entity_presence_get(sdi_entity_hdl_t hdl, bool *pres);

/**
 * @brief  check if there are any faults in given entity.
 * @param[in] hdl - handle to the entity whose information has to be retrieved.
 * @param[out] fault - true if entity has any fault, false otherwise.
 * @return - @ref t_std_error
 */
t_std_error sdi_entity_fault_status_get(sdi_entity_hdl_t hdl, bool *fault);

/**
 * @brief   Get the psu output power status for a given psu.
 * @param[in] entity_hdl - handle to the psu entity whose information has to be
 *                         retrieved.
 * @param[out] status    - true if psu output status is good , false otherwise.
 * @return - @ref t_std_error
 */
t_std_error sdi_entity_psu_output_power_status_get(sdi_entity_hdl_t entity_hdl,
                                                   bool *status);

/**
 * @brief  Retrieve number of resources of given type within given entity.
 * @param[in] hdl handle to the entity whose information has to be retrieved.
 * @param[in] resource_type - type of resource. Example, temperature, fan etc.
 * @return - returns the number of entities of the specified type that are
 *           supported on this system.
 */
uint_t sdi_entity_resource_count_get(sdi_entity_hdl_t hdl,
        sdi_resource_type_t resource_type);
/**
 * @brief retrieve the handle of the resource whose name is known.
 * @param[in] hdl handle to the entity whose information has to be retrieved.
 * @param[in] resource - The type of resource that needs to be looked up.
 * @param[in] alias - the name of the alias. example, "BOOT_STATUS" led.
 * @return - if a resource maching the criteria is found, returns handle to it.
 *           else returns NULL.
 */
sdi_resource_hdl_t sdi_entity_resource_lookup(sdi_entity_hdl_t hdl,
        sdi_resource_type_t resource, const char *alias);

/**
 * @brief - apply the specified function for every resource in the entity
 * for every entity in the sytem, user-specified function "fn" will be called with
 * corresponding entity and user-specified data ar arguments.
 *
 * @param[in] hdl - Handle to the entity whose resources have to be accessed.
 * @param[in] fn - The function that would be called for each resource in the entity.
 * @param[in] user_data - User data that will be passed to the function
 */
void sdi_entity_for_each_resource(sdi_entity_hdl_t hdl,
        void (*fn)(sdi_resource_hdl_t, void *user_data), void *user_data);

/**
 * @brief retrieve the type of the resource for a given resource.
 * @param[in] hdl handle to the resource whose type has to be retrieved.
 * @return - type of the resource @ref sdi_resource_type_t
 */
sdi_resource_type_t sdi_resource_type_get(sdi_resource_hdl_t hdl);

/**
 * Retrieve the alias name of the given resource.
 * resource_hdl[in] - handle to the resource whose name has to be retrieved.
 * return  - the alias name of the resource. example, "BOOT_STATUS" led.
 * else returns NULL.
 */
const char * sdi_resource_alias_get(sdi_resource_hdl_t resource_hdl);
/**
 * @brief retrieve the handle of first resource of the specified type within the entity.
 * @param[in] hdl handle to the entity whose information has to be retrieved.
 * @param[in] resource - The type of resource that needs to be looked up.
 * @return - if a resource maching the criteria is found, returns handle to it.
 *           else returns NULL.
 */
sdi_resource_hdl_t sdi_entity_get_first_resource(sdi_entity_hdl_t hdl,
                                                 sdi_resource_type_t resource);
/**
 * @brief retrieve the handle of next resource of the specified type within the entity.
 * @param[in] hdl handle to the entity whose information has to be retrieved.
 * @param[in] resource - The type of resource that needs to be looked up.
 * @return - if a resource maching the criteria is found, returns handle to it.
 *           else returns NULL.
 */
sdi_resource_hdl_t sdi_entity_get_next_resource(sdi_resource_hdl_t hdl,
                                                sdi_resource_type_t resource);

/**
 * @}
 */

#ifdef __cplusplus
}
#endif

#endif   /* __SDI_ENTITY_H_ */
