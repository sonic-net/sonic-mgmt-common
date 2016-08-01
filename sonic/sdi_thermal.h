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
 * filename: sdi_thermal.h
 */



/**
 * @file sdi_thermal.h
 * @brief SDI Thermal API.
 *
 */

#ifndef __SDI_THERMAL_H_
#define __SDI_THERMAL_H_

#include "std_error_codes.h"
#include "std_type_defs.h"
#include "sdi_entity.h"

/**
 * @defgroup sdi_temperature_api SDI Temperature API.
 * Temperature related API. All Temperature related API take an argument
 * of type sdi_resource_hdl_t.  Application should first identify the
 * right sdi_resource_hdl_t by using @ref sdi_entity_resource_lookup
 *
 * @note unit of temperature is in "Celsius"
 *
 * @ingroup sdi_sys
 * @{
 */

/**
 * @brief Retrieve the temperature using the specified sensor.
 * @param[in] sensor_hdl - handle of the temperature sensor  that is of interest.
 * @param[out] *temp - the value of temperature will be returned in this.
 * @return - standard @ref t_std_error
 */
t_std_error sdi_temperature_get(sdi_resource_hdl_t sensor_hdl, int *temp);

/**
 * @brief Retrieve the threshold set for the specified temperature sensor.
 * @param[in] sensor_hdl - handle to the temperature sensor.
 * @param[in] threshold_type - The type of threhold of interest.
 * @param[out] *val - the value of temperature threshold will be returned in this.
 * @return - standard @ref t_std_error
 */
t_std_error sdi_temperature_threshold_get(sdi_resource_hdl_t sensor_hdl,
        sdi_threshold_t threshold_type,  int *val);

/**
 * @brief Retrieve the temperature using the specified sensor.
 * @param[in] sensor_hdl - handle to the temperature sensor.
 * @param[in] threshold_type - The type of threshold that needs to be set.
 * @param[in] val - the value to be set as threhold for the specified
 *                   temperature resource.
 * @return - standard @ref t_std_error
 */
t_std_error sdi_temperature_threshold_set(sdi_resource_hdl_t sensor_hdl,
        sdi_threshold_t threshold_type, int val);

/**
 * @brief Retrieve the temperature using the specified sensor.
 * @param[in] sensor_hdl - handle to the temperature sensor.
 * @param[out] *alert_on - api returns if this sensor is unhealthy(TRUE) or
 *                       healthy(false).
 * @return - standard @ref t_std_error
 */
t_std_error sdi_temperature_status_get(sdi_resource_hdl_t sensor_hdl, bool *alert_on);

/**
 * @}
 */


#endif   /* __SDI_THERMAL_H_ */
