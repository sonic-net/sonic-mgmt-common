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
 * filename: sdi_entity_info.h
 */



/**
 * @file sdi_entity_info.h
 * @brief Public APIs for reading the entity information.
 * the entity is a physically removable component(eg: system board,
 * PSU , Fantray) which may contain eeprom or other fru that contains
 * information about the entity.
 */

#ifndef __SDI_ENTITY_INFO_H__
#define __SDI_ENTITY_INFO_H__

#include "std_error_codes.h"
#include "std_type_defs.h"
#include "sdi_entity.h"
#include <limits.h>

#ifdef __cplusplus
extern "C" {
#endif

/**
 * @def SDI_MAC_ADDR_LEN
 * Length of MAC address string
 */
#define SDI_MAC_ADDR_LEN 6
/**
 * @def SDI_HW_REV_LEN
 * Length of HW Revision string
 */
#define SDI_HW_REV_LEN   8

/**
 * @def SDI_PPID_LEN
 * Length of PPID string
 */
#define SDI_PPID_LEN  120
#define SDI_PART_NUM_LEN   (10)

/**
 * @defgroup sdi_entity_info_api SDI  ENTITY INFO API.
 * All entity info related API take an argument
 * of type sdi_resource_hdl_t.  Application should first identify the
 * right sdi_resource_hdl_t by using @ref sdi_entity_resource_lookup
 *
 * @ingroup sdi_sys
 * @{
 */


/**
 * @struct sdi_power_type_t
 * supported power types
 */
typedef struct sdi_power_type
{
    int ac_power:1; /**<flag for ac power support*/
    int dc_power:1; /**<flag for dc power support*/
}sdi_power_type_t;

/**
 * @enum sdi_air_flow_type_t
 * supported airflow types
 */
typedef enum{
    /**
     * Air flow direction is Normal = 0
     */
    SDI_PWR_AIR_FLOW_NORMAL,
    /**
     * Air flow direction is Reverse = 1
     */
    SDI_PWR_AIR_FLOW_REVERSE
}sdi_air_flow_type_t;

/**
 * @struct sdi_entity_info_t
 * entity info data structure common for any type of entity
 */
typedef struct {
    char prod_name[NAME_MAX]; /**<null terminated string for name of the product*/
    char ppid[SDI_PPID_LEN]; /**<Dell PPID for a component*/
    char hw_revision[SDI_HW_REV_LEN]; /**<version of the hardware device*/
    char platform_name[NAME_MAX]; /**<name of the platform*/
    char vendor_name[NAME_MAX]; /**<Name of the component vendor*/
    char service_tag[NAME_MAX]; /**<Service tag of the component */
    int mac_size; /**<no.of mac address, the value zero of this,indicates that this
                    entity does not have any associated mac addresses*/
    uint8_t base_mac[SDI_MAC_ADDR_LEN]; /**<base MAC address and this information is specific to system*/
    int num_fans; /**<No.of fans, if this is zero,the entiry doesn't have fans.*/
    int max_speed; /**<maximum speed of the fan*/
    sdi_air_flow_type_t air_flow; /**<air flow direction for the fan*/
    int power_rating; /**<power rating of the device in volt this is applicable only
                        for the power devices.if this is zero, the entity doesn't have any
                        power devices*/
    sdi_power_type_t power_type; /**<type of power*/
    char part_number[SDI_PART_NUM_LEN]; /**<Part number of the hardware device*/
} sdi_entity_info_t;

/**
 * @brief Read the entity info. This api should be called, if entity is present and
 * fault status is not faulty.
 * @param[in] resource_hdl - handle of the entity info resource that is of interest.
 * @param[out] entity_info - the entity info will be returned in this.
 * @return - standard @ref t_std_error
 */
t_std_error sdi_entity_info_read(sdi_resource_hdl_t resource_hdl, sdi_entity_info_t *entity_info);

/**
 * @}
 */

#ifdef __cplusplus
}
#endif
#endif   /* __SDI_ENTITY_INFO_H_ */
