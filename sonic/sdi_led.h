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
 * filename: sdi_led.h
 */


/**
 * @file sdi_led.h
 * @brief SDI LED API.
 *
 */

#ifndef __SDI_LED_H_
#define __SDI_LED_H_

#include "std_error_codes.h"
#include "std_type_defs.h"
#include "sdi_entity.h"

#ifdef __cplusplus
extern "C" {
#endif

/**
 * @defgroup sdi_led_api SDI LED API.
 * LED related API
 * @ingroup sdi_sys
 * @{
 */

/**
 * @defgroup simple_led simple on/off led.
 * @{
 */

/**
 * @brief Turn-on the specified LED
 * @param[in] led_hdl - handle of the led.
 * @return - standard @ref t_std_error
 */
t_std_error sdi_led_on(sdi_resource_hdl_t led_hdl);

/**
 * @brief Turn-off the specified LED
 * @param[in] led_hdl - handle of the led.
 * @return - standard @ref t_std_error
 */
t_std_error sdi_led_off(sdi_resource_hdl_t led_hdl);

/**
 * @}
 */

/**
 * @defgroup digit_display_led LEDs that can display numbers.
 * @{
 */

/**
 * @brief set the specified value in the digital_display_led.
 * @param[in] led_hdl  - handle of led
 * @param[in] display_string - The value to be displayed.
 * @return - STD_ERR_OK on succes and standard error codes on failure. Possible
 * error codes are for this API are as follows.
 * EOPNOTSUPP - If operation not supported by driver
 * EINVAL - If the display_string is not a valid one
 * for all other errors, refer @ref t_std_error
 */
t_std_error sdi_digital_display_led_set(sdi_resource_hdl_t led_hdl, const char *display_string);

/**
 * @brief Turn-on the digital_display_led. Before turn-on the LED, user MUST set
 * the LED with valid string.
 * @param[in] led_hdl  - handle of led
 * @return - standard @ref t_std_error
 */
t_std_error sdi_digital_display_led_on(sdi_resource_hdl_t led_hdl);

/**
 * @brief Turn-off the digital_display_led.
 * @param[in] led_hdl - handle of led
 * @return - standard @ref t_std_error
 */
t_std_error sdi_digital_display_led_off(sdi_resource_hdl_t led_hdl);

/**
 * @}
 */


/**
 * @}
 */

#ifdef __cplusplus
}
#endif

#endif   /* __SDI_LED_H_ */
