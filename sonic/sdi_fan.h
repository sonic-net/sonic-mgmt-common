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
 * filename: sdi_fan.h
 */



/**
 * @file sdi_fan.h
 * @brief SDI FAN API.
 *
 */

#ifndef __SDI_FAN_H_
#define __SDI_FAN_H_

#include "std_error_codes.h"
#include "std_type_defs.h"
#include "sdi_entity.h"

#ifdef __cplusplus
extern "C" {
#endif

/**
 * @defgroup sdi_fan_api SDI FAN API.
 * FAN related API
 * @ingroup sdi_sys
 * @{
 */

/**
 * @brief Retrieve the speed of specified fan.
 * @param[in] fan_hdl - Handle to the fan resource whose speed has to be determined.
 * @param[out] *speed - the speed of the fan in RPM will be returned in this pointer.
 * @return - standard @ref t_std_error
 */
t_std_error sdi_fan_speed_get(sdi_resource_hdl_t fan_hdl, uint_t *speed);

/**
 * @brief Set the speed of specified fan in terms of RPM.
 * @param[in] fan_hdl - Handle to the fan resource whose speed has to be determined.
 * @param[in] speed - the value of speed(in RPM) that needs to be set
 * @return - standard @ref t_std_error
 */
t_std_error sdi_fan_speed_set(sdi_resource_hdl_t fan_hdl, uint_t speed);

/**
 * @brief Retrieve the health of given fan.
 * @param[in] fan_hdl - Handle to the fan resource whose speed has to be determined.
 * @param[out] *alert_on - return value, indicating if fan has any  alert
 * @return - standard @ref t_std_error
 */
t_std_error sdi_fan_status_get(sdi_resource_hdl_t fan_hdl, bool *alert_on);

/**
 * @}
 */

#ifdef __cplusplus
}
#endif
#endif   /* __SDI_FAN_H_ */
